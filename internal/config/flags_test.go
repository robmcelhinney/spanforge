package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFromFlagsWithYAML(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "spanforge.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
rate: 42
rate_unit: traces
rate_interval: 2s
duration: 15s
profile: grpc
routes: 4
services: 5
depth: 3
fanout: 1.4
service_prefix: app-
p50: 20ms
p95: 80ms
p99: 120ms
errors: 1%
retries: 2%
db_heavy: 15%
cache_hit_rate: 75%
variety: high
format: jsonl
output: stdout
flush_interval: 150ms
batch_size: 64
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	flags := FlagValues{
		ConfigFile:       cfgPath,
		Rate:             200,
		RateUnit:         "spans",
		RateInterval:     time.Second,
		Duration:         30 * time.Second,
		Workers:          1,
		Profile:          "web",
		Routes:           8,
		Services:         8,
		Depth:            4,
		Fanout:           2.0,
		ServicePrefix:    "svc-",
		P50:              30 * time.Millisecond,
		P95:              120 * time.Millisecond,
		P99:              350 * time.Millisecond,
		Errors:           "0.5%",
		Retries:          "1%",
		DBHeavy:          "20%",
		CacheHitRate:     "85%",
		Variety:          "medium",
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        512,
		FlushInterval:    200 * time.Millisecond,
		SinkRetries:      2,
		SinkRetryBackoff: 300 * time.Millisecond,
		SinkTimeout:      10 * time.Second,
		SinkMaxInFlight:  2,
	}

	cfg, err := FromFlagsWithOverrides(flags, nil)
	if err != nil {
		t.Fatalf("FromFlagsWithOverrides: %v", err)
	}
	if cfg.RateValue != 42 {
		t.Fatalf("rate=%v want 42", cfg.RateValue)
	}
	if cfg.RateUnit != RateUnitTraces {
		t.Fatalf("rate-unit=%v want traces", cfg.RateUnit)
	}
	if cfg.Profile != "grpc" {
		t.Fatalf("profile=%q want grpc", cfg.Profile)
	}
	if cfg.Variety != "high" {
		t.Fatalf("variety=%q want high", cfg.Variety)
	}
	if cfg.BatchSize != 64 {
		t.Fatalf("batch-size=%d want 64", cfg.BatchSize)
	}
}

func TestFromFlagsWithYAMLCLIOverrides(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "spanforge.yaml")
	if err := os.WriteFile(cfgPath, []byte("services: 12\nprofile: queue\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	flags := FlagValues{
		ConfigFile:       cfgPath,
		Rate:             200,
		RateUnit:         "spans",
		RateInterval:     time.Second,
		Duration:         30 * time.Second,
		Workers:          1,
		Profile:          "web",
		Routes:           8,
		Services:         6,
		Depth:            4,
		Fanout:           2,
		ServicePrefix:    "svc-",
		P50:              30 * time.Millisecond,
		P95:              120 * time.Millisecond,
		P99:              350 * time.Millisecond,
		Errors:           "0.5%",
		Retries:          "1%",
		DBHeavy:          "20%",
		CacheHitRate:     "85%",
		Variety:          "medium",
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        512,
		FlushInterval:    200 * time.Millisecond,
		SinkRetries:      2,
		SinkRetryBackoff: 300 * time.Millisecond,
		SinkTimeout:      10 * time.Second,
		SinkMaxInFlight:  2,
		HighCardinality:  false,
	}

	cfg, err := FromFlagsWithOverrides(flags, map[string]bool{"services": true})
	if err != nil {
		t.Fatalf("FromFlagsWithOverrides: %v", err)
	}
	if cfg.Services != 6 {
		t.Fatalf("services=%d want 6", cfg.Services)
	}
	if cfg.Profile != "queue" {
		t.Fatalf("profile=%q want queue", cfg.Profile)
	}
}

func TestFromFlagsEnvOverridesYAMLAndCLIOverridesEnv(t *testing.T) {
	t.Setenv("SPANFORGE_PROFILE", "queue")
	t.Setenv("SPANFORGE_SERVICES", "11")

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "spanforge.yaml")
	if err := os.WriteFile(cfgPath, []byte("profile: batch\nservices: 7\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	flags := FlagValues{
		ConfigFile:       cfgPath,
		Rate:             200,
		RateUnit:         "spans",
		RateInterval:     time.Second,
		Duration:         30 * time.Second,
		Workers:          1,
		Profile:          "web",
		Routes:           8,
		Services:         5,
		Depth:            4,
		Fanout:           2,
		ServicePrefix:    "svc-",
		P50:              30 * time.Millisecond,
		P95:              120 * time.Millisecond,
		P99:              350 * time.Millisecond,
		Errors:           "0.5%",
		Retries:          "1%",
		DBHeavy:          "20%",
		CacheHitRate:     "85%",
		Variety:          "medium",
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        512,
		FlushInterval:    200 * time.Millisecond,
		SinkRetries:      2,
		SinkRetryBackoff: 300 * time.Millisecond,
		SinkTimeout:      10 * time.Second,
		SinkMaxInFlight:  2,
		HighCardinality:  false,
	}

	cfg, err := FromFlagsWithOverrides(flags, map[string]bool{"services": true})
	if err != nil {
		t.Fatalf("FromFlagsWithOverrides: %v", err)
	}
	if cfg.Profile != "queue" {
		t.Fatalf("profile=%q want queue", cfg.Profile)
	}
	if cfg.Services != 5 {
		t.Fatalf("services=%d want 5", cfg.Services)
	}
}

func TestFromFlagsWithYAMLDebug(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "spanforge.yaml")
	if err := os.WriteFile(cfgPath, []byte("debug: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	flags := FlagValues{
		ConfigFile:       cfgPath,
		Rate:             200,
		RateUnit:         "spans",
		RateInterval:     time.Second,
		Duration:         30 * time.Second,
		Workers:          1,
		Profile:          "web",
		Routes:           8,
		Services:         5,
		Depth:            4,
		Fanout:           2,
		ServicePrefix:    "svc-",
		P50:              30 * time.Millisecond,
		P95:              120 * time.Millisecond,
		P99:              350 * time.Millisecond,
		Errors:           "0.5%",
		Retries:          "1%",
		DBHeavy:          "20%",
		CacheHitRate:     "85%",
		Variety:          "medium",
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        512,
		FlushInterval:    200 * time.Millisecond,
		SinkRetries:      2,
		SinkRetryBackoff: 300 * time.Millisecond,
		SinkTimeout:      10 * time.Second,
		SinkMaxInFlight:  2,
	}

	cfg, err := FromFlagsWithOverrides(flags, nil)
	if err != nil {
		t.Fatalf("FromFlagsWithOverrides: %v", err)
	}
	if !cfg.Debug {
		t.Fatal("expected debug=true from YAML")
	}
}

func TestFromFlagsEnvDebug(t *testing.T) {
	t.Setenv("SPANFORGE_DEBUG", "true")
	flags := FlagValues{
		Rate:             200,
		RateUnit:         "spans",
		RateInterval:     time.Second,
		Duration:         30 * time.Second,
		Workers:          1,
		Profile:          "web",
		Routes:           8,
		Services:         5,
		Depth:            4,
		Fanout:           2,
		ServicePrefix:    "svc-",
		P50:              30 * time.Millisecond,
		P95:              120 * time.Millisecond,
		P99:              350 * time.Millisecond,
		Errors:           "0.5%",
		Retries:          "1%",
		DBHeavy:          "20%",
		CacheHitRate:     "85%",
		Variety:          "medium",
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        512,
		FlushInterval:    200 * time.Millisecond,
		SinkRetries:      2,
		SinkRetryBackoff: 300 * time.Millisecond,
		SinkTimeout:      10 * time.Second,
		SinkMaxInFlight:  2,
	}

	cfg, err := FromFlagsWithOverrides(flags, nil)
	if err != nil {
		t.Fatalf("FromFlagsWithOverrides: %v", err)
	}
	if !cfg.Debug {
		t.Fatal("expected debug=true from env")
	}
}
