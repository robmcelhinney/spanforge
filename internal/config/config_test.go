package config

import "testing"

func TestParseRate(t *testing.T) {
	tests := []struct {
		in       string
		wantUnit RateUnit
	}{
		{in: "spans", wantUnit: RateUnitSpans},
		{in: "traces", wantUnit: RateUnitTraces},
	}

	for _, tc := range tests {
		gotUnit, err := ParseRateUnit(tc.in)
		if err != nil {
			t.Fatalf("ParseRateUnit(%q): %v", tc.in, err)
		}
		if gotUnit != tc.wantUnit {
			t.Fatalf("ParseRateUnit(%q) unit=%q want=%q", tc.in, gotUnit, tc.wantUnit)
		}
	}
}

func TestParsePercent(t *testing.T) {
	got, err := ParsePercent("0.5%")
	if err != nil {
		t.Fatalf("ParsePercent: %v", err)
	}
	if got != 0.005 {
		t.Fatalf("ParsePercent got %v want 0.005", got)
	}
}

func TestValidateRequiresOTLPEndpoint(t *testing.T) {
	cfg := Config{
		RateValue:        1,
		RateUnit:         RateUnitSpans,
		RateInterval:     1,
		Duration:         1,
		Workers:          1,
		Profile:          "web",
		Routes:           1,
		Services:         1,
		Depth:            1,
		Fanout:           1,
		P50:              1,
		P95:              2,
		P99:              3,
		Errors:           0,
		Retries:          0,
		CacheHitRate:     1,
		Format:           "otlp-http",
		Output:           "otlp",
		BatchSize:        1,
		FlushInterval:    1,
		SinkRetries:      0,
		SinkRetryBackoff: 1,
		SinkTimeout:      1,
		SinkMaxInFlight:  1,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateFormatOutputPairing(t *testing.T) {
	cfg := Config{
		RateValue:        1,
		RateUnit:         RateUnitSpans,
		RateInterval:     1,
		Duration:         1,
		Workers:          1,
		Profile:          "web",
		Routes:           1,
		Services:         1,
		Depth:            1,
		Fanout:           1,
		P50:              1,
		P95:              2,
		P99:              3,
		Errors:           0,
		Retries:          0,
		CacheHitRate:     1,
		Format:           "pretty",
		Output:           "file",
		BatchSize:        1,
		FlushInterval:    1,
		SinkRetries:      0,
		SinkRetryBackoff: 1,
		SinkTimeout:      1,
		SinkMaxInFlight:  1,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for pretty+file")
	}
}

func TestValidateVariety(t *testing.T) {
	cfg := Config{
		RateValue:        1,
		RateUnit:         RateUnitSpans,
		RateInterval:     1,
		Duration:         1,
		Workers:          1,
		Profile:          "web",
		Routes:           1,
		Services:         1,
		Depth:            1,
		Fanout:           1,
		P50:              1,
		P95:              2,
		P99:              3,
		Errors:           0,
		Retries:          0,
		CacheHitRate:     1,
		Variety:          "extreme",
		Format:           "jsonl",
		Output:           "stdout",
		BatchSize:        1,
		FlushInterval:    1,
		SinkRetries:      0,
		SinkRetryBackoff: 1,
		SinkTimeout:      1,
		SinkMaxInFlight:  1,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid variety error")
	}
}

func TestValidateNoopOutput(t *testing.T) {
	cfg := Config{
		RateValue:        1,
		RateUnit:         RateUnitSpans,
		RateInterval:     1,
		Duration:         1,
		Workers:          1,
		Profile:          "web",
		Routes:           1,
		Services:         1,
		Depth:            1,
		Fanout:           1,
		P50:              1,
		P95:              2,
		P99:              3,
		Errors:           0,
		Retries:          0,
		CacheHitRate:     1,
		Variety:          "medium",
		Format:           "otlp-http",
		Output:           "noop",
		BatchSize:        1,
		FlushInterval:    1,
		SinkRetries:      0,
		SinkRetryBackoff: 1,
		SinkTimeout:      1,
		SinkMaxInFlight:  1,
		HighCardinality:  false,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected noop to validate: %v", err)
	}
}
