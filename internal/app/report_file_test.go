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
	if got["run_id"] != "sf_seed_1" {
		t.Fatalf("run_id=%v want sf_seed_1", got["run_id"])
	}
	if got["profile"] != "web" {
		t.Fatalf("profile=%v want web", got["profile"])
	}
	if services, ok := got["services"].([]any); !ok || len(services) == 0 {
		t.Fatalf("missing services: %s", string(data))
	}
	if traceIDs, ok := got["sample_trace_ids"].([]any); !ok || len(traceIDs) == 0 {
		t.Fatalf("missing sample_trace_ids: %s", string(data))
	}
}

func TestRunPhaseFileAddsPhaseReport(t *testing.T) {
	tmp := t.TempDir()
	phasePath := filepath.Join(tmp, "phases.yaml")
	reportPath := filepath.Join(tmp, "report.json")
	if err := os.WriteFile(phasePath, []byte(`
phases:
  - name: warmup
    duration: 1s
    rate: 10
    rate_unit: traces
    errors: 0%
  - name: spike
    duration: 1s
    rate: 20
    rate_unit: traces
    errors: 5%
    p95: 100ms
    p99: 150ms
`), 0o644); err != nil {
		t.Fatalf("write phase file: %v", err)
	}
	cfg := reportTestConfig(reportPath)
	cfg.Count = 4
	cfg.PhaseFile = phasePath

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run: %v", err)
	}

	report := readReport(t, reportPath)
	phases, ok := report["phases"].([]any)
	if !ok || len(phases) != 2 {
		t.Fatalf("phases=%v want two phases", report["phases"])
	}
	first := phases[0].(map[string]any)
	second := phases[1].(map[string]any)
	if first["name"] != "warmup" || second["name"] != "spike" {
		t.Fatalf("phase names=%v, %v want warmup, spike", first["name"], second["name"])
	}
}

func TestRunBuiltInLoadAddsPhaseReport(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "report.json")
	cfg := reportTestConfig(reportPath)
	cfg.Count = 3
	cfg.Load = "warmup-spike-recovery"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run: %v", err)
	}

	report := readReport(t, reportPath)
	phases, ok := report["phases"].([]any)
	if !ok || len(phases) != 3 {
		t.Fatalf("phases=%v want three phases", report["phases"])
	}
	traceIDs, ok := report["sample_trace_ids"].([]any)
	if !ok || len(traceIDs) != 3 {
		t.Fatalf("sample_trace_ids=%v want three distinct trace samples", report["sample_trace_ids"])
	}
}

func reportTestConfig(reportPath string) config.Config {
	return config.Config{
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
}

func readReport(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json parse report: %v", err)
	}
	return got
}
