package types

import (
	"testing"
	"time"
)

func TestContextMarksDurationsOnce(t *testing.T) {
	start := time.Unix(100, 0)
	c := &Context{StartTime: start}

	c.MarkFirstToken(start.Add(1250 * time.Millisecond))
	c.MarkFirstToken(start.Add(2 * time.Second))
	c.MarkLatency(start.Add(3*time.Second + 400*time.Millisecond))
	c.MarkLatency(start.Add(5 * time.Second))

	if c.FirstTokenMs != 1250 {
		t.Fatalf("first token ms = %d, want 1250", c.FirstTokenMs)
	}
	if c.LatencyMs != 3400 {
		t.Fatalf("latency ms = %d, want 3400", c.LatencyMs)
	}
}
