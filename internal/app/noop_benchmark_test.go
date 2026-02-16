package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
)

func TestRunNoopBenchmarkOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := config.Config{
		RateValue:        100,
		RateUnit:         config.RateUnitTraces,
		RateInterval:     time.Second,
		Duration:         time.Second,
		Count:            3,
		Seed:             1,
		Workers:          1,
		Profile:          "web",
		Routes:           3,
		Services:         3,
		Depth:            2,
		Fanout:           1,
		ServicePrefix:    "svc-",
		P50:              10 * time.Millisecond,
		P95:              50 * time.Millisecond,
		P99:              80 * time.Millisecond,
		Errors:           0,
		Retries:          0,
		DBHeavy:          0,
		CacheHitRate:     1,
		Variety:          "low",
		Format:           "otlp-http",
		Output:           "noop",
		BatchSize:        32,
		FlushInterval:    100 * time.Millisecond,
		SinkRetries:      0,
		SinkRetryBackoff: 100 * time.Millisecond,
		SinkTimeout:      time.Second,
		SinkMaxInFlight:  1,
		HighCardinality:  false,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := Run(cfg, buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "benchmark summary:") {
		t.Fatalf("missing benchmark output: %q", out)
	}
	if !strings.Contains(out, "spans/sec=") {
		t.Fatalf("missing spans/sec: %q", out)
	}
}
