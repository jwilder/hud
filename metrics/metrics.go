package metrics

import (
	"sync"
)

type Metric interface {
	Name() string
	Value() interface{}
	Reset()
	Snapshot() Metric
}

type Counter struct {
	sync.RWMutex
	v    int64
	name string
	ts   int64
}

type Collection struct {
	sync.Mutex
	metrics map[string]Metric
}

func NewCollection() *Collection {
	return &Collection{
		metrics: make(map[string]Metric),
	}
}

func (m *Collection) Reset() {
	m.Lock()
	defer m.Unlock()
	for _, v := range m.metrics {
		if _, ok := v.(*Counter); ok {
			v.Reset()
		}
	}
}

func (m *Collection) Snapshot() *Collection {
	snap := NewCollection()
	for k, v := range m.metrics {
		snap.metrics[k] = v.Snapshot()
	}
	return snap
}

func (m *Collection) Metrics() map[string]Metric {
	return m.metrics
}

func (m *Collection) GetOrRegisterCounter(name string) *Counter {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := m.metrics[name]; ok {
		return c.(*Counter)
	}
	c := NewCounter(name)
	m.metrics[name] = c
	return c
}

func (m *Collection) GetOrRegisterGauge(name string) *Gauge {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := m.metrics[name]; ok {
		return c.(*Gauge)
	}
	c := NewGauge(name)
	m.metrics[name] = c
	return c
}

func (m *Collection) GetOrRegisterGaugeFloat64(name string) *GaugeFloat64 {
	mu.Lock()
	defer mu.Unlock()
	if c, ok := m.metrics[name]; ok {
		return c.(*GaugeFloat64)
	}
	c := NewGaugeFloat64(name)
	m.metrics[name] = c
	return c
}

func NewCounter(name string) *Counter {
	return &Counter{
		name: name,
	}
}

func (c *Counter) Snapshot() Metric {
	snap := NewCounter(c.name)
	snap.v = c.v
	return snap
}

func (c *Counter) Reset() {
	c.Lock()
	defer c.Unlock()
	c.v = 0
}

func (c *Counter) Inc(value int64) {
	c.Lock()
	defer c.Unlock()
	c.v += value
}

func (c *Counter) Name() string {
	return c.name
}

func (c *Counter) Value() interface{} {
	c.Lock()
	defer c.Unlock()
	return c.v
}

type Gauge struct {
	sync.Mutex
	v    int64
	name string
}

func NewGauge(name string) *Gauge {
	return &Gauge{
		name: name,
	}
}

func (c *Gauge) Snapshot() Metric {
	snap := NewGauge(c.name)
	snap.v = c.v
	return snap
}

func (c *Gauge) Reset() {
	c.Lock()
	defer c.Unlock()
	c.v = 0
}

func (c *Gauge) Set(value int64) {
	c.Lock()
	c.v = value
	c.Unlock()
}

func (c *Gauge) Value() interface{} {
	c.Lock()
	defer c.Unlock()

	return c.v
}

func (c *Gauge) Name() string {
	return c.name
}

type GaugeFloat64 struct {
	sync.Mutex
	v    float64
	name string
}

func NewGaugeFloat64(name string) *GaugeFloat64 {
	return &GaugeFloat64{
		name: name,
	}
}

func (c *GaugeFloat64) Snapshot() Metric {
	snap := NewGaugeFloat64(c.name)
	snap.v = c.v
	return snap
}

func (c *GaugeFloat64) Reset() {
	c.Lock()
	defer c.Unlock()
	c.v = 0
}

func (c *GaugeFloat64) Set(value float64) {
	c.Lock()
	defer c.Unlock()
	c.v = value
}

func (c *GaugeFloat64) Value() interface{} {
	c.Lock()
	defer c.Unlock()
	return c.v
}

func (c *GaugeFloat64) Name() string {
	return c.name
}
