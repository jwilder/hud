package metrics

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	mu         sync.Mutex
	metrics    = NewCollection()
	MetricChan = make(chan Metric)
	handlers   []chan *Collection
)

type Handler interface {
	SendForever(metrics chan *Collection)
}

func GetOrRegisterCounter(name string) *Counter {
	return metrics.GetOrRegisterCounter(name)
}

func GetOrRegisterGauge(name string) *Gauge {
	return metrics.GetOrRegisterGauge(name)
}

func GetOrRegisterGaugeFloat64(name string) *GaugeFloat64 {
	return metrics.GetOrRegisterGaugeFloat64(name)
}

func AddHandler(handler Handler) {
	mu.Lock()
	defer mu.Unlock()
	sendChan := make(chan *Collection)
	handlers = append(handlers, sendChan)
	go handler.SendForever(sendChan)
}

func FlushPeriodically(interval int) {
	for {
		mu.Lock()

		snap := metrics.Snapshot()
		for _, handler := range handlers {
			select {
			case handler <- snap:
			default:
				log.Warn("Full listener.  Metrics not flushed.")
			}
		}
		//metrics = NewCollection()
		metrics.Reset()
		mu.Unlock()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

type Collector struct {
	Prefix string
}

func (c *Collector) RecordGauge(name string, value int64) {
	metric := GetOrRegisterGauge(c.metricName(name))
	metric.Set(value)
}

func (c *Collector) RecordGaugeFloat64(name string, value float64) {
	metric := GetOrRegisterGaugeFloat64(c.metricName(name))
	metric.Set(value)
}

func (c *Collector) RecordCount(name string, value int64) {
	metric := GetOrRegisterCounter(c.metricName(name))
	metric.Inc(value)
}

func (c *Collector) metricName(name string) string {
	if c.Prefix != "" {
		return c.Prefix + "." + name
	}
	return name
}

func (c *Collector) Prefixed(name string) string {
	return c.metricName(name)
}
