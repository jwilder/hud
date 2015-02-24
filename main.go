package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"
	log "github.com/Sirupsen/logrus"
	"github.com/jwilder/hud/docker"
	"github.com/jwilder/hud/host"
	"github.com/jwilder/hud/logger"
	"github.com/jwilder/hud/metrics"
)

var (
	statsPrefix     string
	debug           bool
	version         bool
	buildVersion    string
	influxDBAddr    string
	influxDBUser    string
	influxDBPass    string
	influxDBDB      string
	graphiteAddr    string
	hostname        string
	noStats         bool
	flushInterval   int
	httpClient      *http.Client
	logDests        sliceVar
	logFmts         sliceVar
	noLogs          bool
	wg              sync.WaitGroup
	logDestinations []logDestination
)

type logDestination struct {
	dest   string
	format string
}

type sliceVar []string

func (s *sliceVar) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *sliceVar) String() string {
	return strings.Join(*s, ",")
}

func parseLogDestinations(logDests, logFmts sliceVar) ([]logDestination, error) {
	if len(logDests) == 0 {
		logDests = sliceVar{"console"}
	}

	dests := []logDestination{}
	for _, dest := range logDests {
		format := "short"
		parts := strings.Split(dest, "=")
		if len(parts) == 2 {
			format = parts[1]
		}
		addr := parts[0]
		if addr != "console" {
			u, err := url.Parse(addr)
			if err != nil {
				log.Fatalf("ERROR: Bad log-to addr: %s", err)
			}
			switch u.Scheme {
			case "udp", "tcp", "tls":
				break
			default:
				log.Fatalf("ERROR: Unsupported log-to addr: %s", addr)
			}
		}

		dests = append(dests,
			logDestination{
				dest:   addr,
				format: format,
			})
	}
	return dests, nil
}

func main() {
	flag.StringVar(&statsPrefix, "prefix", "", "Global prefix for all stats")
	flag.BoolVar(&debug, "debug", false, "Enables debug logging")
	flag.BoolVar(&version, "v", false, "Display version info")
	flag.BoolVar(&noLogs, "no-logs", false, "Disable log tailing")
	flag.BoolVar(&noStats, "no-stats", false, "Disable stats collection")
	flag.IntVar(&flushInterval, "flush-interval", 60, "Flush metrics every interval seconds")

	flag.StringVar(&influxDBAddr, "influxdb-addr", "", "InfluxDB host:port")
	flag.StringVar(&influxDBUser, "influxdb-user", "", "InfluxDB username")
	flag.StringVar(&influxDBPass, "influxdb-pass", "", "InfluxDB password")
	flag.StringVar(&influxDBDB, "influxdb-db", "", "InfluxDB database")
	flag.StringVar(&graphiteAddr, "graphite-addr", "", "Graphite host:port")
	flag.StringVar(&hostname, "hostname", "", "Hostname of this host for remote logging systems")
	flag.Var(&logDests, "log-to", "Log destination and format [console, [tcp|udp|tls://]host:port][=short,ext,json,syslog]. (default console)")

	flag.Parse()

	if version {
		fmt.Println(buildVersion)
		return
	}

	log.SetOutput(os.Stderr)

	endpoint, err := docker.GetEndpoint()
	if err != nil {
		log.Fatalf("Bad docker endpoint: %s", err)
	}

	if debug {
		log.Debug("Debug enabled")
		log.SetLevel(log.DebugLevel)
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if graphiteAddr != "" && !noStats {
		log.Infof("Sending metrics to graphite at %s", graphiteAddr)
		g, err := metrics.NewGraphite(graphiteAddr, 3*time.Second)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		metrics.AddHandler(g)
	}

	if influxDBAddr != "" && !noStats {
		log.Infof("Sending metrics to influxdb at %s", influxDBAddr)
		i, err := metrics.NewInfluxDB(influxDBUser, influxDBPass, influxDBAddr, influxDBDB)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}

		metrics.AddHandler(i)
	}

	broadcaster := &docker.Broadcaster{
		Endpoint: endpoint,
	}
	wg.Add(1)

	dockerC := docker.NewDockerCollector(statsPrefix, broadcaster, flushInterval)
	if !noStats {
		wg.Add(1)
		go metrics.FlushPeriodically(flushInterval)

		hostC := host.NewHostCollector(statsPrefix, flushInterval)
		wg.Add(1)
		go hostC.CollectForever()

		wg.Add(1)
		go dockerC.CollectForever()
	}

	if hostname == "" {
		hostname, err = os.Hostname()
		if err != nil {
			log.Fatalf("ERROR: Unable to lookup hostname: %s", err)
		}
	}

	if !noLogs {
		logDests, err := parseLogDestinations(logDests, logFmts)
		if err != nil {
			log.Fatalf("ERROR: Unable to parse log destinations: %s", err)
		}

		for _, dest := range logDests {
			var f logger.Formatter
			f = &logger.ShortFormatter{}
			switch dest.format {
			case "ext":
				f = &logger.ExtendedFormatter{}
			case "json":
				f = &logger.JSONFormatter{}
			case "syslog":

				f = &logger.SyslogFormatter{
					Hostname: hostname,
					Severity: logger.SevInfo,
					Facility: logger.LogLocal1,
					Newline:  strings.HasPrefix(dest.dest, "tcp://"),
				}
			}

			switch dest.dest {
			case "console":
				f.SetColored(true)
				cl, err := logger.NewConsoleLogger(os.Stdout, f)
				if err != nil {
					log.Fatalf("ERROR: %s", err)
				}
				dockerC.AddLogHandler(cl)
			default:
				f.SetColored(false)
				sl, err := logger.NewSocketLogger(dest.dest, nil, f)
				if err != nil {
					log.Fatalf("ERROR: %s", err)
				}
				dockerC.AddLogHandler(sl)
			}
		}
	}
	go broadcaster.WatchForever()

	wg.Wait()
}
