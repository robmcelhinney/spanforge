package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type FlagValues struct {
	ConfigFile       string
	Rate             float64
	RateUnit         string
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
	Errors           string
	Retries          string
	DBHeavy          string
	CacheHitRate     string
	Variety          string
	HighCardinality  bool
	Format           string
	Output           string
	File             string
	OTLPEndpoint     string
	ZipkinEndpoint   string
	OTLPInsecure     bool
	Headers          []string
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

type yamlFlagValues struct {
	Rate             *float64 `yaml:"rate"`
	RateUnit         *string  `yaml:"rate_unit"`
	RateInterval     *string  `yaml:"rate_interval"`
	Duration         *string  `yaml:"duration"`
	Count            *int     `yaml:"count"`
	Seed             *int64   `yaml:"seed"`
	Workers          *int     `yaml:"workers"`
	Profile          *string  `yaml:"profile"`
	Routes           *int     `yaml:"routes"`
	Services         *int     `yaml:"services"`
	Depth            *int     `yaml:"depth"`
	Fanout           *float64 `yaml:"fanout"`
	ServicePrefix    *string  `yaml:"service_prefix"`
	P50              *string  `yaml:"p50"`
	P95              *string  `yaml:"p95"`
	P99              *string  `yaml:"p99"`
	Errors           *string  `yaml:"errors"`
	Retries          *string  `yaml:"retries"`
	DBHeavy          *string  `yaml:"db_heavy"`
	CacheHitRate     *string  `yaml:"cache_hit_rate"`
	Variety          *string  `yaml:"variety"`
	HighCardinality  *bool    `yaml:"high_cardinality"`
	Format           *string  `yaml:"format"`
	Output           *string  `yaml:"output"`
	File             *string  `yaml:"file"`
	OTLPEndpoint     *string  `yaml:"otlp_endpoint"`
	ZipkinEndpoint   *string  `yaml:"zipkin_endpoint"`
	OTLPInsecure     *bool    `yaml:"otlp_insecure"`
	Headers          []string `yaml:"headers"`
	Compress         *string  `yaml:"compress"`
	BatchSize        *int     `yaml:"batch_size"`
	FlushInterval    *string  `yaml:"flush_interval"`
	SinkRetries      *int     `yaml:"sink_retries"`
	SinkRetryBackoff *string  `yaml:"sink_retry_backoff"`
	SinkTimeout      *string  `yaml:"sink_timeout"`
	SinkMaxInFlight  *int     `yaml:"sink_max_in_flight"`
	ReportFile       *string  `yaml:"report_file"`
	HTTPListen       *string  `yaml:"http_listen"`
	Debug            *bool    `yaml:"debug"`
}

func AddFlags(fs *pflag.FlagSet, v *FlagValues) {
	fs.StringVar(&v.ConfigFile, "config", "", "Path to YAML config file")
	fs.Float64Var(&v.Rate, "rate", 200, "Generation rate amount")
	fs.StringVar(&v.RateUnit, "rate-unit", "spans", "Rate unit: spans or traces")
	fs.DurationVar(&v.RateInterval, "rate-interval", 1*time.Second, "Time interval for rate amount")
	fs.DurationVar(&v.Duration, "duration", 30*time.Second, "Run duration (set to 0s for no time limit)")
	fs.IntVar(&v.Count, "count", 0, "Total span/trace count (overrides duration if > 0)")
	fs.Int64Var(&v.Seed, "seed", 1, "Random seed")
	fs.IntVar(&v.Workers, "workers", 1, "Concurrent generator workers")
	fs.StringVar(&v.Profile, "profile", "web", "Generation profile")
	fs.IntVar(&v.Routes, "routes", 8, "Number of named routes/methods per profile")
	fs.IntVar(&v.Services, "services", 8, "Number of services")
	fs.IntVar(&v.Depth, "depth", 4, "Max trace depth")
	fs.Float64Var(&v.Fanout, "fanout", 2.0, "Average span fanout")
	fs.StringVar(&v.ServicePrefix, "service-prefix", "svc-", "Service name prefix")
	fs.DurationVar(&v.P50, "p50", 30*time.Millisecond, "p50 span latency")
	fs.DurationVar(&v.P95, "p95", 120*time.Millisecond, "p95 span latency")
	fs.DurationVar(&v.P99, "p99", 350*time.Millisecond, "p99 span latency")
	fs.StringVar(&v.Errors, "errors", "0.5%", "Error rate percentage")
	fs.StringVar(&v.Retries, "retries", "1%", "Retry rate percentage")
	fs.StringVar(&v.DBHeavy, "db-heavy", "20%", "DB-intensive operation ratio")
	fs.StringVar(&v.CacheHitRate, "cache-hit-rate", "85%", "Cache hit ratio")
	fs.StringVar(&v.Variety, "variety", "medium", "Variety level: low, medium, high")
	fs.BoolVar(&v.HighCardinality, "high-cardinality", false, "Enable high-cardinality attributes (request IDs, message IDs)")
	fs.StringVar(&v.Format, "format", "jsonl", "Output format")
	fs.StringVar(&v.Output, "output", "stdout", "Output sink")
	fs.StringVar(&v.File, "file", "", "Output file path")
	fs.StringVar(&v.OTLPEndpoint, "otlp-endpoint", "", "OTLP endpoint")
	fs.StringVar(&v.ZipkinEndpoint, "zipkin-endpoint", "", "Zipkin endpoint")
	fs.BoolVar(&v.OTLPInsecure, "otlp-insecure", true, "Use insecure OTLP gRPC transport")
	fs.StringSliceVar(&v.Headers, "headers", nil, "Additional headers (repeat k=v)")
	fs.StringVar(&v.Compress, "compress", "", "Compression for OTLP HTTP (gzip)")
	fs.IntVar(&v.BatchSize, "batch-size", 512, "Spans per batch")
	fs.DurationVar(&v.FlushInterval, "flush-interval", 200*time.Millisecond, "Sink flush interval")
	fs.IntVar(&v.SinkRetries, "sink-retries", 2, "Retry attempts for sink requests")
	fs.DurationVar(&v.SinkRetryBackoff, "sink-retry-backoff", 300*time.Millisecond, "Backoff between sink retries")
	fs.DurationVar(&v.SinkTimeout, "sink-timeout", 10*time.Second, "Per-request sink timeout")
	fs.IntVar(&v.SinkMaxInFlight, "sink-max-in-flight", 2, "Maximum concurrent in-flight sink requests")
	fs.StringVar(&v.ReportFile, "report-file", "", "Write run summary as JSON to this path")
	fs.StringVar(&v.HTTPListen, "http-listen", "127.0.0.1:8080", "Admin HTTP listen address for /healthz and /stats")
	fs.BoolVar(&v.Debug, "debug", false, "Enable debug logs for trace emission and sink sends")
}

func FromFlags(v FlagValues) (Config, error) {
	return FromFlagsWithOverrides(v, nil)
}

func FromFlagsWithOverrides(v FlagValues, cliOverrides map[string]bool) (Config, error) {
	if (cliOverrides == nil || !cliOverrides["config"]) && v.ConfigFile == "" {
		if raw, ok := os.LookupEnv("SPANFORGE_CONFIG"); ok {
			v.ConfigFile = strings.TrimSpace(raw)
		}
	}

	if v.ConfigFile != "" {
		merged, err := mergeFromYAML(v, cliOverrides)
		if err != nil {
			return Config{}, err
		}
		v = merged
	}

	mergedEnv, err := mergeFromEnv(v, cliOverrides)
	if err != nil {
		return Config{}, err
	}
	v = mergedEnv

	rateUnit, err := ParseRateUnit(v.RateUnit)
	if err != nil {
		return Config{}, err
	}
	errorsRate, err := ParsePercent(v.Errors)
	if err != nil {
		return Config{}, err
	}
	retriesRate, err := ParsePercent(v.Retries)
	if err != nil {
		return Config{}, err
	}
	dbHeavyRate, err := ParsePercent(v.DBHeavy)
	if err != nil {
		return Config{}, err
	}
	cacheHitRate, err := ParsePercent(v.CacheHitRate)
	if err != nil {
		return Config{}, err
	}
	headers, err := ParseHeaders(v.Headers)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		RateValue:        v.Rate,
		RateUnit:         rateUnit,
		RateInterval:     v.RateInterval,
		Duration:         v.Duration,
		Count:            v.Count,
		Seed:             v.Seed,
		Workers:          v.Workers,
		Profile:          v.Profile,
		Routes:           v.Routes,
		Services:         v.Services,
		Depth:            v.Depth,
		Fanout:           v.Fanout,
		ServicePrefix:    v.ServicePrefix,
		P50:              v.P50,
		P95:              v.P95,
		P99:              v.P99,
		Errors:           errorsRate,
		Retries:          retriesRate,
		DBHeavy:          dbHeavyRate,
		CacheHitRate:     cacheHitRate,
		Variety:          v.Variety,
		HighCardinality:  v.HighCardinality,
		Format:           v.Format,
		Output:           v.Output,
		File:             v.File,
		OTLPEndpoint:     v.OTLPEndpoint,
		ZipkinEndpoint:   v.ZipkinEndpoint,
		OTLPInsecure:     v.OTLPInsecure,
		Headers:          headers,
		Compress:         v.Compress,
		BatchSize:        v.BatchSize,
		FlushInterval:    v.FlushInterval,
		SinkRetries:      v.SinkRetries,
		SinkRetryBackoff: v.SinkRetryBackoff,
		SinkTimeout:      v.SinkTimeout,
		SinkMaxInFlight:  v.SinkMaxInFlight,
		ReportFile:       v.ReportFile,
		HTTPListen:       v.HTTPListen,
		Debug:            v.Debug,
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func mergeFromYAML(v FlagValues, cliOverrides map[string]bool) (FlagValues, error) {
	data, err := os.ReadFile(v.ConfigFile)
	if err != nil {
		return FlagValues{}, fmt.Errorf("read config file: %w", err)
	}
	var y yamlFlagValues
	if err := yaml.Unmarshal(data, &y); err != nil {
		return FlagValues{}, fmt.Errorf("parse config yaml: %w", err)
	}
	overridden := func(name string) bool {
		if cliOverrides == nil {
			return false
		}
		return cliOverrides[name]
	}
	setString := func(flag string, src *string, dst *string) {
		if src != nil && !overridden(flag) {
			*dst = *src
		}
	}
	setInt := func(flag string, src *int, dst *int) {
		if src != nil && !overridden(flag) {
			*dst = *src
		}
	}
	setInt64 := func(flag string, src *int64, dst *int64) {
		if src != nil && !overridden(flag) {
			*dst = *src
		}
	}
	setFloat := func(flag string, src *float64, dst *float64) {
		if src != nil && !overridden(flag) {
			*dst = *src
		}
	}
	setBool := func(flag string, src *bool, dst *bool) {
		if src != nil && !overridden(flag) {
			*dst = *src
		}
	}
	setDuration := func(flag string, src *string, dst *time.Duration) error {
		if src == nil || overridden(flag) {
			return nil
		}
		d, err := time.ParseDuration(*src)
		if err != nil {
			return fmt.Errorf("invalid %s duration %q: %w", flag, *src, err)
		}
		*dst = d
		return nil
	}

	setFloat("rate", y.Rate, &v.Rate)
	setString("rate-unit", y.RateUnit, &v.RateUnit)
	if err := setDuration("rate-interval", y.RateInterval, &v.RateInterval); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("duration", y.Duration, &v.Duration); err != nil {
		return FlagValues{}, err
	}
	setInt("count", y.Count, &v.Count)
	setInt64("seed", y.Seed, &v.Seed)
	setInt("workers", y.Workers, &v.Workers)
	setString("profile", y.Profile, &v.Profile)
	setInt("routes", y.Routes, &v.Routes)
	setInt("services", y.Services, &v.Services)
	setInt("depth", y.Depth, &v.Depth)
	setFloat("fanout", y.Fanout, &v.Fanout)
	setString("service-prefix", y.ServicePrefix, &v.ServicePrefix)
	if err := setDuration("p50", y.P50, &v.P50); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("p95", y.P95, &v.P95); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("p99", y.P99, &v.P99); err != nil {
		return FlagValues{}, err
	}
	setString("errors", y.Errors, &v.Errors)
	setString("retries", y.Retries, &v.Retries)
	setString("db-heavy", y.DBHeavy, &v.DBHeavy)
	setString("cache-hit-rate", y.CacheHitRate, &v.CacheHitRate)
	setString("variety", y.Variety, &v.Variety)
	setBool("high-cardinality", y.HighCardinality, &v.HighCardinality)
	setString("format", y.Format, &v.Format)
	setString("output", y.Output, &v.Output)
	setString("file", y.File, &v.File)
	setString("otlp-endpoint", y.OTLPEndpoint, &v.OTLPEndpoint)
	setString("zipkin-endpoint", y.ZipkinEndpoint, &v.ZipkinEndpoint)
	setBool("otlp-insecure", y.OTLPInsecure, &v.OTLPInsecure)
	if len(y.Headers) > 0 && !overridden("headers") {
		v.Headers = append([]string(nil), y.Headers...)
	}
	setString("compress", y.Compress, &v.Compress)
	setInt("batch-size", y.BatchSize, &v.BatchSize)
	if err := setDuration("flush-interval", y.FlushInterval, &v.FlushInterval); err != nil {
		return FlagValues{}, err
	}
	setInt("sink-retries", y.SinkRetries, &v.SinkRetries)
	if err := setDuration("sink-retry-backoff", y.SinkRetryBackoff, &v.SinkRetryBackoff); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("sink-timeout", y.SinkTimeout, &v.SinkTimeout); err != nil {
		return FlagValues{}, err
	}
	setInt("sink-max-in-flight", y.SinkMaxInFlight, &v.SinkMaxInFlight)
	setString("report-file", y.ReportFile, &v.ReportFile)
	setString("http-listen", y.HTTPListen, &v.HTTPListen)
	setBool("debug", y.Debug, &v.Debug)

	return v, nil
}

func mergeFromEnv(v FlagValues, cliOverrides map[string]bool) (FlagValues, error) {
	overridden := func(name string) bool {
		if cliOverrides == nil {
			return false
		}
		return cliOverrides[name]
	}

	setString := func(flag, env string, dst *string) {
		if overridden(flag) {
			return
		}
		if raw, ok := os.LookupEnv(env); ok {
			*dst = raw
		}
	}
	setInt := func(flag, env string, dst *int) error {
		if overridden(flag) {
			return nil
		}
		raw, ok := os.LookupEnv(env)
		if !ok {
			return nil
		}
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("invalid %s=%q: %w", env, raw, err)
		}
		*dst = n
		return nil
	}
	setInt64 := func(flag, env string, dst *int64) error {
		if overridden(flag) {
			return nil
		}
		raw, ok := os.LookupEnv(env)
		if !ok {
			return nil
		}
		n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid %s=%q: %w", env, raw, err)
		}
		*dst = n
		return nil
	}
	setFloat := func(flag, env string, dst *float64) error {
		if overridden(flag) {
			return nil
		}
		raw, ok := os.LookupEnv(env)
		if !ok {
			return nil
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return fmt.Errorf("invalid %s=%q: %w", env, raw, err)
		}
		*dst = n
		return nil
	}
	setBool := func(flag, env string, dst *bool) error {
		if overridden(flag) {
			return nil
		}
		raw, ok := os.LookupEnv(env)
		if !ok {
			return nil
		}
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("invalid %s=%q: %w", env, raw, err)
		}
		*dst = b
		return nil
	}
	setDuration := func(flag, env string, dst *time.Duration) error {
		if overridden(flag) {
			return nil
		}
		raw, ok := os.LookupEnv(env)
		if !ok {
			return nil
		}
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("invalid %s=%q: %w", env, raw, err)
		}
		*dst = d
		return nil
	}

	if err := setFloat("rate", "SPANFORGE_RATE", &v.Rate); err != nil {
		return FlagValues{}, err
	}
	setString("rate-unit", "SPANFORGE_RATE_UNIT", &v.RateUnit)
	if err := setDuration("rate-interval", "SPANFORGE_RATE_INTERVAL", &v.RateInterval); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("duration", "SPANFORGE_DURATION", &v.Duration); err != nil {
		return FlagValues{}, err
	}
	if err := setInt("count", "SPANFORGE_COUNT", &v.Count); err != nil {
		return FlagValues{}, err
	}
	if err := setInt64("seed", "SPANFORGE_SEED", &v.Seed); err != nil {
		return FlagValues{}, err
	}
	if err := setInt("workers", "SPANFORGE_WORKERS", &v.Workers); err != nil {
		return FlagValues{}, err
	}
	setString("profile", "SPANFORGE_PROFILE", &v.Profile)
	if err := setInt("routes", "SPANFORGE_ROUTES", &v.Routes); err != nil {
		return FlagValues{}, err
	}
	if err := setInt("services", "SPANFORGE_SERVICES", &v.Services); err != nil {
		return FlagValues{}, err
	}
	if err := setInt("depth", "SPANFORGE_DEPTH", &v.Depth); err != nil {
		return FlagValues{}, err
	}
	if err := setFloat("fanout", "SPANFORGE_FANOUT", &v.Fanout); err != nil {
		return FlagValues{}, err
	}
	setString("service-prefix", "SPANFORGE_SERVICE_PREFIX", &v.ServicePrefix)
	if err := setDuration("p50", "SPANFORGE_P50", &v.P50); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("p95", "SPANFORGE_P95", &v.P95); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("p99", "SPANFORGE_P99", &v.P99); err != nil {
		return FlagValues{}, err
	}
	setString("errors", "SPANFORGE_ERRORS", &v.Errors)
	setString("retries", "SPANFORGE_RETRIES", &v.Retries)
	setString("db-heavy", "SPANFORGE_DB_HEAVY", &v.DBHeavy)
	setString("cache-hit-rate", "SPANFORGE_CACHE_HIT_RATE", &v.CacheHitRate)
	setString("variety", "SPANFORGE_VARIETY", &v.Variety)
	if err := setBool("high-cardinality", "SPANFORGE_HIGH_CARDINALITY", &v.HighCardinality); err != nil {
		return FlagValues{}, err
	}
	setString("format", "SPANFORGE_FORMAT", &v.Format)
	setString("output", "SPANFORGE_OUTPUT", &v.Output)
	setString("file", "SPANFORGE_FILE", &v.File)
	setString("otlp-endpoint", "SPANFORGE_OTLP_ENDPOINT", &v.OTLPEndpoint)
	setString("zipkin-endpoint", "SPANFORGE_ZIPKIN_ENDPOINT", &v.ZipkinEndpoint)
	if err := setBool("otlp-insecure", "SPANFORGE_OTLP_INSECURE", &v.OTLPInsecure); err != nil {
		return FlagValues{}, err
	}
	if !overridden("headers") {
		if raw, ok := os.LookupEnv("SPANFORGE_HEADERS"); ok && strings.TrimSpace(raw) != "" {
			parts := strings.Split(raw, ",")
			v.Headers = v.Headers[:0]
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					v.Headers = append(v.Headers, trimmed)
				}
			}
		}
	}
	setString("compress", "SPANFORGE_COMPRESS", &v.Compress)
	if err := setInt("batch-size", "SPANFORGE_BATCH_SIZE", &v.BatchSize); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("flush-interval", "SPANFORGE_FLUSH_INTERVAL", &v.FlushInterval); err != nil {
		return FlagValues{}, err
	}
	if err := setInt("sink-retries", "SPANFORGE_SINK_RETRIES", &v.SinkRetries); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("sink-retry-backoff", "SPANFORGE_SINK_RETRY_BACKOFF", &v.SinkRetryBackoff); err != nil {
		return FlagValues{}, err
	}
	if err := setDuration("sink-timeout", "SPANFORGE_SINK_TIMEOUT", &v.SinkTimeout); err != nil {
		return FlagValues{}, err
	}
	if err := setInt("sink-max-in-flight", "SPANFORGE_SINK_MAX_IN_FLIGHT", &v.SinkMaxInFlight); err != nil {
		return FlagValues{}, err
	}
	setString("report-file", "SPANFORGE_REPORT_FILE", &v.ReportFile)
	setString("http-listen", "SPANFORGE_HTTP_LISTEN", &v.HTTPListen)
	if err := setBool("debug", "SPANFORGE_DEBUG", &v.Debug); err != nil {
		return FlagValues{}, err
	}

	return v, nil
}
