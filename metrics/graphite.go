package metrics

import (
	"fmt"
	//"log"
	"net"
	"strings"
	"time"
	log "github.com/Sirupsen/logrus"
)

// Graphite represents a Graphite server. You Register expvars
// in this struct, which will be published to the server on a
// regular interval.
type Graphite struct {
	endpoint   string
	interval   time.Duration
	timeout    time.Duration
	connection net.Conn
	shutdown   chan chan bool
}

// NewGraphite returns a Graphite structure with an open and working
// connection, but no active/registered variables being published.
// Endpoint should be of the format "host:port", eg. "stats:2003".
// Interval is the (best-effort) minimum duration between (sequential)
// publishments of Registered expvars. Timeout is per-publish-action.
func NewGraphite(endpoint string, timeout time.Duration) (*Graphite, error) {
	g := &Graphite{
		endpoint:   endpoint,
		timeout:    timeout,
		connection: nil,
		shutdown:   make(chan chan bool),
	}
	if err := g.reconnect(); err != nil {
		return nil, err
	}
	return g, nil
}

// Shutdown signals the Graphite structure to stop publishing
// Registered expvars.
func (g *Graphite) Shutdown() {
	q := make(chan bool)
	g.shutdown <- q
	<-q
}

func (g *Graphite) SendForever(metrics chan *Collection) {
	for {
		select {
		case collection := <-metrics:
			m := collection.Metrics()
			for name, metric := range m {
				val := metric.Value()
				strVal := "0"
				switch val := val.(type) {
				case int64:
					strVal = fmt.Sprintf("%d", val)
				case float64:
					strVal = fmt.Sprintf("%0.2f", val)
				default:
					log.Errorf("Unhandled graphite type: %s", val)
				}

				if err := g.sendOne(strings.Replace(name, "/", "_", -1), strVal); err != nil {
					log.Errorf("ERROR: %s: %s", name, err)
				}
			}
		case q := <-g.shutdown:
			g.connection.Close()
			g.connection = nil
			q <- true
			return
		}
	}
}

// sendOne publishes the given name-value pair to the Graphite server.
// If the connection is broken, one reconnect attempt is made.
func (g *Graphite) sendOne(name, value string) error {
	if g.connection == nil {
		if err := g.reconnect(); err != nil {
			return fmt.Errorf("failed; reconnect attempt: %s", err)
		}
	}
	deadline := time.Now().Add(g.timeout)
	if err := g.connection.SetWriteDeadline(deadline); err != nil {
		g.connection = nil
		return fmt.Errorf("SetWriteDeadline: %s", err)
	}
	p := fmt.Sprintf("%s %s %d", name, value, time.Now().Unix())
	b := []byte(p + "\n")
	log.Debug(p)

	if n, err := g.connection.Write([]byte(b)); err != nil {
		g.connection = nil
		return fmt.Errorf("Write: %s", err)
	} else if n != len(b) {
		g.connection = nil
		return fmt.Errorf("%s = %v: short write: %d/%d", name, value, n, len(b))
	}
	return nil
}

// reconnect attempts to (re-)establish a TCP connection to the Graphite server.
func (g *Graphite) reconnect() error {
	conn, err := net.Dial("tcp", g.endpoint)
	if err != nil {
		return err
	}
	g.connection = conn
	return nil
}
