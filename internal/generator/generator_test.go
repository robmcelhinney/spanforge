package generator

import (
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
)

func baseConfig() config.Config {
	return config.Config{
		RateValue:        100,
		RateUnit:         config.RateUnitSpans,
		RateInterval:     time.Second,
		Duration:         10 * time.Second,
		Count:            0,
		Seed:             42,
		Workers:          1,
		Profile:          "web",
		Routes:           5,
		Services:         4,
		Depth:            3,
		Fanout:           1.5,
		ServicePrefix:    "svc-",
		P50:              10 * time.Millisecond,
		P95:              100 * time.Millisecond,
		P99:              200 * time.Millisecond,
		Errors:           0.1,
		Retries:          0.2,
		DBHeavy:          0.2,
		CacheHitRate:     0.8,
		Variety:          "medium",
		HighCardinality:  false,
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        256,
		FlushInterval:    100 * time.Millisecond,
		SinkRetries:      0,
		SinkRetryBackoff: 100 * time.Millisecond,
		SinkTimeout:      time.Second,
		SinkMaxInFlight:  1,
	}
}

func TestDeterministicWithSameSeed(t *testing.T) {
	cfg := baseConfig()
	now := time.Unix(1700000000, 0).UTC()

	g1 := New(cfg)
	g2 := New(cfg)

	t1 := g1.GenerateTrace(now)
	t2 := g2.GenerateTrace(now)

	if t1.TraceID != t2.TraceID {
		t.Fatal("trace IDs differ for same seed")
	}
	if len(t1.Spans) != len(t2.Spans) {
		t.Fatalf("span count mismatch: %d vs %d", len(t1.Spans), len(t2.Spans))
	}
	if len(t1.Spans) == 0 {
		t.Fatal("expected at least one span")
	}
	if t1.Spans[0].SpanID != t2.Spans[0].SpanID {
		t.Fatal("first span ID differs for same seed")
	}
}

func TestRootSpanIsServer(t *testing.T) {
	cfg := baseConfig()
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	if len(trace.Spans) == 0 {
		t.Fatal("expected spans")
	}
	if trace.Spans[0].Kind != "SERVER" {
		t.Fatalf("root span kind=%q want SERVER", trace.Spans[0].Kind)
	}
}
