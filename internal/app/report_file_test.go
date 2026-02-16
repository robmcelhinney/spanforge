package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
)

func TestRunWritesReportFile(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "report.json")
	cfg := config.Config{
		RateValue:        100,
		RateUnit:         config.RateUnitTraces,
		RateInterval:     time.Second,
		Duration:         time.Second,
		Count:            3,
		Seed:             1,
		Workers:          1,
		Profile:          "web",
		Routes:           2,
		Services:         2,
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
		HighCardinality:  false,
		Format:           "otlp-http",
		Output:           "noop",
		BatchSize:        32,
		FlushInterval:    100 * time.Millisecond,
		SinkRetries:      0,
		SinkRetryBackoff: 100 * time.Millisecond,
		SinkTimeout:      time.Second,
		SinkMaxInFlight:  1,
		ReportFile:       reportPath,
		HTTPListen:       "",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json parse report: %v", err)
	}
	if got["spans_per_second"] == nil {
		t.Fatalf("missing spans_per_second: %s", string(data))
	}
}
