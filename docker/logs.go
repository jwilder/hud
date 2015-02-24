package docker

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	dockerapi "github.com/fsouza/go-dockerclient"
)

type LogChannel chan *LogRecord

type Tailer struct {
	sync.Mutex
	Broadcaster *Broadcaster
	watchers    map[string]bool
	Running     bool
	logHandlers []LogChannel
}

type LogRecord struct {
	Ts            time.Time
	ContainerID   string
	ContainerName string
	Stream        string
	Message       string
}

type LogHandler interface {
	HandleLog(log *LogRecord) error
}

func (t *Tailer) watchContainer(client *dockerapi.Client, id string, all bool) error {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.watchers[id]; ok {
		return nil
	}
	t.watchers[id] = true

	go func() {
		t.tailContainer(client, id, all)
		t.Lock()
		defer t.Unlock()
		delete(t.watchers, id)
	}()
	//t.watchers[id] = watcher
	return nil
}

func (t *Tailer) AddLogHandler(h LogHandler) {
	t.Lock()
	defer t.Unlock()
	logChan := make(LogChannel, 100)
	t.logHandlers = append(t.logHandlers, logChan)
	go t.handleLogs(logChan, h)
}

func (t *Tailer) RemoveLogHandler(c LogChannel) {
	t.Lock()
	defer t.Unlock()

	logHandlers := []LogChannel{}
	for _, l := range t.logHandlers {
		if &l == &c {
			continue
		}
		logHandlers = append(logHandlers, l)
	}
	t.logHandlers = logHandlers
}

func (t *Tailer) notifyLog(msg *LogRecord) {
	t.Lock()
	defer t.Unlock()
	for _, c := range t.logHandlers {
		c <- msg
	}
}

func (t *Tailer) handleLogs(logs LogChannel, handler LogHandler) {
	for {
		log := <-logs
		handler.HandleLog(log)
	}
}

func (t *Tailer) onWatch(client *dockerapi.Client) {
	containers, err := client.ListContainers(dockerapi.ListContainersOptions{
		All:  false,
		Size: false,
	})
	if err != nil {
		log.Errorf("ERROR: %s", err)
	}

	for _, container := range containers {
		err := t.watchContainer(client, container.ID, false)
		if err != nil {
			log.Errorf("ERROR: %s", err)
		}
	}
}

func (t *Tailer) onEvent(client *dockerapi.Client, event *dockerapi.APIEvents) {
	log.Debugf("Received event %s for container %s", event.Status, event.ID[:12])
	if event.Status == "create" {
		err := t.watchContainer(client, event.ID, true)
		if err != nil {
			log.Errorf("ERROR: %s", err)
		}
	}

	if event.Status == "start" {
		err := t.watchContainer(client, event.ID, false)
		if err != nil {
			log.Errorf("ERROR: %s", err)
		}
	}
}

func (t *Tailer) Tail() {
	t.watchers = map[string]bool{}
	t.Broadcaster.AddPreWatchHandler(t.onWatch)
	t.Broadcaster.AddEventHandler(t.onEvent)
}

func (t *Tailer) tailContainer(client *dockerapi.Client, id string, all bool) {
	container, err := client.InspectContainer(id)
	if err != nil {
		log.Errorf("ERROR: tailing: %s\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	success := make(chan struct{})
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	go t.WriteLogs(&NamedReader{
		Name:   strings.TrimPrefix(container.Name, "/"),
		ID:     container.ID,
		Reader: stdoutReader,
		Stream: "stdout"})
	go t.WriteLogs(&NamedReader{
		Name:   strings.TrimPrefix(container.Name, "/"),
		ID:     container.ID,
		Reader: stderrReader,
		Stream: "stderr"})

	go func() {
		log.Debugf("Attached to container %s", container.ID[0:12])
		defer wg.Done()
		err = client.AttachToContainer(dockerapi.AttachToContainerOptions{
			Container:    container.ID,
			OutputStream: stdoutWriter,
			ErrorStream:  stderrWriter,
			Logs:         all,
			Stdin:        false,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Success:      success,
			RawTerminal:  container.Config.Tty,
		})
		if err != nil {
			close(success)
			log.Errorf("ERROR: Unable to attach to container: %s\n", err)
			return
		}
	}()

	_, ok := <-success
	if ok {
		success <- struct{}{}
	}

	wg.Wait()
	stdoutWriter.Close()
	stderrWriter.Close()
	log.Debugf("Detached from container %s", container.ID[0:12])
}

func (w *Tailer) WriteLogs(input *NamedReader) {
	buf := bufio.NewReaderSize(input, 4096*16)
	for {
		data, err := buf.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				println(err)
			}
			return
		}

		w.notifyLog(&LogRecord{
			Ts:            time.Now(),
			ContainerID:   input.ID,
			ContainerName: input.Name,
			Stream:        input.Stream,
			Message:       string(data),
		})
	}
}

type NamedReader struct {
	Reader io.Reader
	Name   string
	ID     string
	Stream string
}

func (r *NamedReader) Read(p []byte) (n int, err error) {
	return r.Reader.Read(p)
}
