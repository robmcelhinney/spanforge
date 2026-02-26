package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type RateUnit string

const (
	RateUnitSpans  RateUnit = "spans"
	RateUnitTraces RateUnit = "traces"
)

type Config struct {
	RateValue        float64
	RateUnit         RateUnit
	RateInterval     time.Duration
	Duration         time.Duration
	Count            int
	Seed             int64
	Workers          int
	Profile          string
	Routes           int
	Services         int
	Depth            int
	Fanout           float64
	ServicePrefix    string
	P50              time.Duration
	P95              time.Duration
	P99              time.Duration
	Errors           float64
	Retries          float64
	DBHeavy          float64
	CacheHitRate     float64
	Variety          string
	HighCardinality  bool
	Format           string
	Output           string
	File             string
	OTLPEndpoint     string
	ZipkinEndpoint   string
	OTLPInsecure     bool
	Headers          map[string]string
	Compress         string
	BatchSize        int
	FlushInterval    time.Duration
	SinkRetries      int
	SinkRetryBackoff time.Duration
	SinkTimeout      time.Duration
	SinkMaxInFlight  int
	ReportFile       string
	HTTPListen       string
	Debug            bool
}

func ParseRateUnit(raw string) (RateUnit, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(RateUnitSpans):
		return RateUnitSpans, nil
	case string(RateUnitTraces):
		return RateUnitTraces, nil
	default:
		return "", fmt.Errorf("invalid rate-unit %q (must be spans or traces)", raw)
	}
}

func ParsePercent(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	if strings.HasSuffix(s, "%") {
		s = strings.TrimSuffix(s, "%")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid percent %q", raw)
	}
	if v < 0 || v > 100 {
		return 0, fmt.Errorf("percent out of range [0,100]: %q", raw)
	}
	return v / 100.0, nil
}

func ParseHeaders(items []string) (map[string]string, error) {
	out := make(map[string]string, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid header %q (expected k=v)", item)
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out, nil
}

func (c Config) Validate() error {
	if c.RateValue <= 0 {
		return fmt.Errorf("rate must be > 0")
	}
	if c.RateInterval <= 0 {
		return fmt.Errorf("rate-interval must be > 0")
	}
	if c.RateUnit != RateUnitSpans && c.RateUnit != RateUnitTraces {
		return fmt.Errorf("rate-unit must be spans or traces")
	}
	if c.Duration < 0 {
		return fmt.Errorf("duration must be >= 0")
	}
	if c.Count < 0 {
		return fmt.Errorf("count must be >= 0")
	}
	if c.Workers <= 0 {
		return fmt.Errorf("workers must be > 0")
	}
	if c.Services <= 0 {
		return fmt.Errorf("services must be > 0")
	}
	if c.Routes <= 0 {
		return fmt.Errorf("routes must be > 0")
	}
	if c.Depth <= 0 {
		return fmt.Errorf("depth must be > 0")
	}
	if c.Fanout <= 0 {
		return fmt.Errorf("fanout must be > 0")
	}
	if c.P50 <= 0 || c.P95 <= 0 || c.P99 <= 0 {
		return fmt.Errorf("p50/p95/p99 must be > 0")
	}
	if c.P50 > c.P95 || c.P95 > c.P99 {
		return fmt.Errorf("latency percentiles must satisfy p50 <= p95 <= p99")
	}
	if c.Errors < 0 || c.Errors > 1 || c.Retries < 0 || c.Retries > 1 || c.DBHeavy < 0 || c.DBHeavy > 1 || c.CacheHitRate < 0 || c.CacheHitRate > 1 {
		return fmt.Errorf("errors/retries/db-heavy/cache-hit-rate must be in [0,1]")
	}

	switch c.Profile {
	case "web", "grpc", "queue", "batch":
	default:
		return fmt.Errorf("profile must be one of web, grpc, queue, batch")
	}

	switch strings.ToLower(strings.TrimSpace(c.Variety)) {
	case "", "low", "medium", "high":
	default:
		return fmt.Errorf("variety must be one of low, medium, high")
	}

	needsOTLPEndpoint := c.Output == "otlp" || ((c.Format == "otlp-http" || c.Format == "otlp-grpc") && c.Output != "noop")
	if needsOTLPEndpoint && strings.TrimSpace(c.OTLPEndpoint) == "" {
		return fmt.Errorf("otlp endpoint required for output=%q format=%q", c.Output, c.Format)
	}
	needsZipkinEndpoint := c.Output == "zipkin" || (c.Format == "zipkin-json" && c.Output != "noop")
	if needsZipkinEndpoint && strings.TrimSpace(c.ZipkinEndpoint) == "" {
		return fmt.Errorf("zipkin endpoint required for output=%q format=%q", c.Output, c.Format)
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be > 0")
	}
	if c.FlushInterval <= 0 {
		return fmt.Errorf("flush-interval must be > 0")
	}
	if c.SinkRetries < 0 {
		return fmt.Errorf("sink-retries must be >= 0")
	}
	if c.SinkRetryBackoff <= 0 {
		return fmt.Errorf("sink-retry-backoff must be > 0")
	}
	if c.SinkTimeout <= 0 {
		return fmt.Errorf("sink-timeout must be > 0")
	}
	if c.SinkMaxInFlight <= 0 {
		return fmt.Errorf("sink-max-in-flight must be > 0")
	}
	switch c.Format {
	case "jsonl":
		if c.Output != "stdout" && c.Output != "file" && c.Output != "noop" {
			return fmt.Errorf("jsonl format requires output stdout, file, or noop")
		}
	case "pretty":
		if c.Output != "stdout" && c.Output != "noop" {
			return fmt.Errorf("pretty format requires output stdout or noop")
		}
	case "otlp-http":
		if c.Output != "otlp" && c.Output != "noop" {
			return fmt.Errorf("otlp-http format requires output otlp or noop")
		}
	case "otlp-grpc":
		if c.Output != "otlp" && c.Output != "noop" {
			return fmt.Errorf("otlp-grpc format requires output otlp or noop")
		}
	case "zipkin-json":
		if c.Output != "zipkin" && c.Output != "noop" {
			return fmt.Errorf("zipkin-json format requires output zipkin or noop")
		}
	default:
		return fmt.Errorf("unsupported format %q", c.Format)
	}
	switch c.Output {
	case "stdout", "file", "otlp", "zipkin", "noop":
	default:
		return fmt.Errorf("unsupported output %q", c.Output)
	}
	if c.Output == "file" && strings.TrimSpace(c.File) == "" {
		return fmt.Errorf("file output requires --file")
	}

	return nil
}
