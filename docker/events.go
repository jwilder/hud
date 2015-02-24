package docker

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	dockerapi "github.com/fsouza/go-dockerclient"
)

const DefaultReconnectTimeout = 1 * time.Second

type PreWatch func(client *dockerapi.Client)
type EventHandler func(client *dockerapi.Client, event *dockerapi.APIEvents)

type Broadcaster struct {
	sync.Mutex
	Endpoint         string
	eventHandlers    []EventHandler
	preWatchHandlers []PreWatch
}

func (b *Broadcaster) AddEventHandler(fn EventHandler) {
	b.Lock()
	defer b.Unlock()
	b.eventHandlers = append(b.eventHandlers, fn)
}

func (b *Broadcaster) RemoveEventHandler(fn EventHandler) {
	b.Lock()
	defer b.Unlock()

	eventHandlers := []EventHandler{}
	for _, l := range b.eventHandlers {
		if &l == &fn {
			continue
		}
		eventHandlers = append(eventHandlers, l)
	}
	b.eventHandlers = eventHandlers
}

func (b *Broadcaster) AddPreWatchHandler(fn PreWatch) {
	b.Lock()
	defer b.Unlock()
	b.preWatchHandlers = append(b.preWatchHandlers, fn)
}

func (b *Broadcaster) RemovePreWatchHandler(fn PreWatch) {
	b.Lock()
	defer b.Unlock()

	handlers := []PreWatch{}
	for _, h := range b.preWatchHandlers {
		if &h == &fn {
			continue
		}
		handlers = append(handlers, h)
	}
	b.preWatchHandlers = handlers
}

func (b *Broadcaster) broadcast(client *dockerapi.Client, event *dockerapi.APIEvents) {
	b.Lock()
	defer b.Unlock()
	for _, fn := range b.eventHandlers {
		// make sure writing on the channel does not block
		go fn(client, event)
	}
}

func (b *Broadcaster) notifyPreWatch(client *dockerapi.Client) {
	b.Lock()
	defer b.Unlock()
	for _, fn := range b.preWatchHandlers {
		// make sure writing on the channel does not block
		go fn(client)
	}
}
func (b *Broadcaster) WatchForever() {
	var client *dockerapi.Client

	for {
		if client == nil {
			var err error
			client, err = NewDockerClient(b.Endpoint)
			if err != nil {
				log.Errorf("Unable to connect to docker daemon: %s", err)
				time.Sleep(DefaultReconnectTimeout)
				continue
			}
		}

		eventChan := make(chan *dockerapi.APIEvents, 100)
		defer close(eventChan)

		watching := false
		for {

			if client == nil {
				break
			}
			err := client.Ping()
			if err != nil {
				log.Errorf("Unable to ping docker daemon: %s", err)
				if watching {
					client.RemoveEventListener(eventChan)
					watching = false
					client = nil
				}
				time.Sleep(DefaultReconnectTimeout)
				break

			}

			if !watching {
				b.notifyPreWatch(client)
				err = client.AddEventListener(eventChan)
				if err != nil && err != dockerapi.ErrListenerAlreadyExists {
					log.Errorf("Error registering docker event listener: %s", err)
					time.Sleep(DefaultReconnectTimeout)
					continue
				}
				watching = true
				log.Debug("Watching docker events")
			}

			select {

			case event := <-eventChan:
				if event == nil {
					if watching {
						client.RemoveEventListener(eventChan)
						watching = false
						client = nil
					}
					break
				}

				b.broadcast(client, event)
			case <-time.After(10 * time.Second):
				// check for docker liveness
			}

		}
	}
}
