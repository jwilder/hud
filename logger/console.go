package logger

import (
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/jwilder/hud/docker"
)

type ConsoleLogger struct {
	w         io.Writer
	formatter Formatter
	logger    *log.Logger
}

func NewConsoleLogger(w io.Writer, formatter Formatter) (*ConsoleLogger, error) {
	logger := log.New()
	logger.Out = w
	return &ConsoleLogger{
		w:         w,
		formatter: formatter,
		logger:    logger,
	}, nil
}

func (l *ConsoleLogger) HandleLog(msg *docker.LogRecord) error {
	line, err := l.formatter.Format(msg)
	if err != nil {
		return err
	}

	if line == nil {
		return nil
	}

	l.w.Write(line)
	return nil
}
