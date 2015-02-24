package metrics

import (
	"testing"
)

func TestCounterAdd(t *testing.T) {
	c := NewCounter("foo")
	c.Inc(1)
	if c.Value().(int64) != 1 {
		t.Fatalf("expected 1, got %d", c.Value())
	}

	for i := 0; i < 10; i++ {
		c.Inc(1)
	}

	if c.Value().(int64) != 11 {
		t.Fatalf("expected 11, got %d", c.Value())
	}

	c.Inc(10)
	if c.Value().(int64) != 21 {
		t.Fatalf("expected 21, got %d", c.Value())
	}
}

func TestGaugeSet(t *testing.T) {
	c := NewGauge("foo")
	c.Set(1)
	if c.Value().(int64) != 1 {
		t.Fatalf("expected 1, got %d", c.Value())
	}

	for i := 0; i < 10; i++ {
		c.Set(int64(i))
	}

	if c.Value().(int64) != 9 {
		t.Fatalf("expected 11, got %d", c.Value())
	}

	c.Set(21)
	if c.Value().(int64) != 21 {
		t.Fatalf("expected 21, got %d", c.Value())
	}
}

func TestGaugeFlat64Set(t *testing.T) {
	c := NewGaugeFloat64("foo")
	c.Set(1)
	if c.Value().(float64) != 1 {
		t.Fatalf("expected 1, got %d", c.Value())
	}

	for i := 0; i < 10; i++ {
		c.Set(float64(i))
	}

	if int(c.Value().(float64)) != 9 {
		t.Fatalf("expected 11, got %d", c.Value())
	}

	c.Set(21)
	if int(c.Value().(float64)) != 21 {
		t.Fatalf("expected 21, got %d", c.Value())
	}
}
