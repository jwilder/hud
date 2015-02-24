package logger

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"
	"unicode"

	log "github.com/Sirupsen/logrus"
	"github.com/jwilder/hud/ansi"
	"github.com/jwilder/hud/docker"
)

const (
	StdDateFormat     = "2006-01-02T15:04:05.999Z"
	Rfc5424DateFormat = "2006-01-02T15:04:05.999999Z07:00"
)

var (
	Colors = []string{
		ansi.ColorGreen,
		ansi.ColorOrange,
		ansi.ColorCyan,
		ansi.ColorYellow,
		ansi.ColorLightBlue,
		ansi.ColorLightPurple,
		ansi.ColorWhite,
	}
	epoch      time.Time
	isTerminal bool
)

type Formatter interface {
	SetColored(colored bool)
	Format(log *docker.LogRecord) ([]byte, error)
}

type JSONFormatter struct{}

func init() {
	isTerminal = log.IsTerminal()
	epoch = time.Now()
}

func miniTS() int {
	return int(time.Since(epoch) / time.Second)
}

func (f *JSONFormatter) Format(log *docker.LogRecord) ([]byte, error) {
	data := map[string]interface{}{}

	data["time"] = log.Ts.UTC().Format(StdDateFormat)
	data["msg"] = log.Message
	data["stream"] = log.Stream
	data["name"] = log.ContainerName
	data["id"] = log.ContainerID

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}

func (f *JSONFormatter) SetColored(colored bool) {}

type ShortFormatter struct {
	colored bool
}

func (f *ShortFormatter) SetColored(colored bool) {
	f.colored = colored
}

func (f *ShortFormatter) hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (f *ShortFormatter) Format(rec *docker.LogRecord) ([]byte, error) {
	msg := strings.TrimRightFunc(rec.Message, unicode.IsSpace)

	if msg == "" {
		return nil, nil
	}

	if isTerminal && f.colored {
		color := int(f.hash(rec.ContainerName)) % len(Colors)

		return []byte(fmt.Sprintf("%s %s: %s\x1b[0m\n",
			f.colorize(fmt.Sprintf("[%04d]", miniTS()), ansi.ColorWhite),
			f.colorize(rec.ContainerName, Colors[color]),
			string(ansi.StripAnsiControl([]byte(msg))))), nil

	}

	return []byte(fmt.Sprintf("[%04d] %s: %s\x1b[0m\n",
		miniTS(),
		rec.ContainerName,
		string(ansi.StripAnsi([]byte(msg))))), nil
}

func (f *ShortFormatter) colorize(text, color string) string {
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", color, text)
}

type ExtendedFormatter struct {
	colored bool
}

func (f *ExtendedFormatter) SetColored(colored bool) {
	f.colored = colored
}

func (f *ExtendedFormatter) hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (f *ExtendedFormatter) Format(rec *docker.LogRecord) ([]byte, error) {
	msg := strings.TrimRightFunc(rec.Message, unicode.IsSpace)

	if msg == "" {
		return nil, nil
	}

	if isTerminal && f.colored {
		color := int(f.hash(rec.ContainerName)) % len(Colors)

		return []byte(fmt.Sprintf("%-24s %s msg=\"%s\"\x1b[0m\n",
			f.colorize(rec.Ts.UTC().Format(StdDateFormat), ansi.ColorWhite),
			f.colorize("container="+rec.ContainerName, Colors[color]),
			string(ansi.StripAnsiControl([]byte(msg))))), nil
	}

	return []byte(fmt.Sprintf("%-24s container=%s msg=\"%s\"\n",
		rec.Ts.UTC().Format(StdDateFormat),
		rec.ContainerName,
		string(ansi.StripAnsi([]byte(msg))))), nil
}

func (f *ExtendedFormatter) colorize(text, color string) string {
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", color, text)
}

type SyslogFormatter struct {
	colored  bool
	Facility Priority
	Severity Priority
	Hostname string
	Newline  bool
}

func (f *SyslogFormatter) SetColored(colored bool) {
}

func (f *SyslogFormatter) priority() Priority {
	return (f.Facility << 3) | f.Severity
}

func (f *SyslogFormatter) Format(rec *docker.LogRecord) ([]byte, error) {
	msg := strings.Replace(rec.Message, "\n", " ", -1)
	msg = strings.Replace(msg, "\r", " ", -1)
	msg = strings.Replace(msg, "\x00", " ", -1)

	if msg == "" {
		return nil, nil
	}

	ts := rec.Ts.Format(Rfc5424DateFormat)
	tag := rec.ContainerName

	buf := fmt.Sprintf("<%d>1 %s %s %s - - - %s", f.priority(), ts, f.Hostname, tag, msg)
	if f.Newline {
		return []byte(buf + "\n"), nil
	}
	return []byte(buf), nil
}
