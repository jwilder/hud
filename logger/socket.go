/*
The syslog package provides a syslog client.
Unlike the core log/syslog package it uses the newer rfc5424 syslog protocol,
reliably reconnects on failure, and supports TLS encrypted TCP connections.
*/
package logger

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"
	"github.com/jwilder/hud/docker"

	"net/url"
	log "github.com/Sirupsen/logrus"
)

// A Logger is a connection to a syslog server. It reconnects on error.
// Clients log by sending a Packet to the logger.Packets channel.
type SocketLogger struct {
	conn      net.Conn
	proto     string
	raddr     string
	rootCAs   *x509.CertPool
	formatter Formatter
}

func NewSocketLogger(dest string, rootCAs *x509.CertPool, formatter Formatter) (*SocketLogger, error) {
	u, err := url.Parse(dest)
	if err != nil {
		return nil, err
	}

	proto := "udp"
	if u.Scheme != "" {
		proto = u.Scheme
	}

	// dial once, just to make sure the network is working
	//conn, err := dial(proto, u.Host, rootCAs)

	if err != nil {
		return nil, err
	}
	logger := &SocketLogger{
		proto:   proto,
		raddr:   u.Host,
		rootCAs: rootCAs,
		//conn:      conn,
		formatter: formatter,
	}
	return logger, nil

}

// dial connects to the server and set up a watching goroutine
func dial(proto, raddr string, rootCAs *x509.CertPool) (net.Conn, error) {
	var netConn net.Conn
	var err error

	switch proto {
	case "tls":
		var config *tls.Config
		if rootCAs != nil {
			config = &tls.Config{RootCAs: rootCAs}
		}
		netConn, err = tls.Dial("tcp", raddr, config)
	case "udp", "tcp":
		netConn, err = net.Dial(proto, raddr)
	default:
		return nil, fmt.Errorf("Network protocol %s not supported", proto)
	}
	if err != nil {
		return nil, err
	}
	return netConn, nil
}

// Connect to the server, retrying every 10 seconds until successful.
func (l *SocketLogger) connect() {
	for {
		c, err := dial(l.proto, l.raddr, l.rootCAs)
		if err == nil {
			l.conn = c
			return
		} else {
			log.Errorf("ERROR: %s", err)
			time.Sleep(10 * time.Second)
		}
	}
}

func (l *SocketLogger) HandleLog(log *docker.LogRecord) error {

	line, err := l.formatter.Format(log)
	if err != nil {
		return err
	}

	if line == nil {
		return nil
	}

	if l.conn == nil {
		l.connect()
	}

	var n int
	switch l.conn.(type) {
	case *net.TCPConn, *tls.Conn, *net.UDPConn:
		n, err = l.conn.Write(line)
	default:
		panic(fmt.Errorf("Network protocol %s not supported", l.proto))
	}

	if err != nil {
		l.conn.Close()
		l.conn = nil
		return err
	}

	if n != len(line) {
		l.conn.Close()
		l.conn = nil
		return fmt.Errorf("short read. expect %d. got %d", n, len(line))
	}

	return nil
}
