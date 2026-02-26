package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
	jsonlenc "github.com/robmcelhinney/spanforge/internal/encode/jsonl"
	prettyenc "github.com/robmcelhinney/spanforge/internal/encode/pretty"
	"github.com/robmcelhinney/spanforge/internal/generator"
	"github.com/robmcelhinney/spanforge/internal/model"
	"github.com/robmcelhinney/spanforge/internal/sink/otlpgrpc"
	"github.com/robmcelhinney/spanforge/internal/sink/otlphttp"
	"github.com/robmcelhinney/spanforge/internal/sink/zipkin"
)

func debugf(cfg config.Config, format string, args ...any) {
	if !cfg.Debug {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "spanforge debug: "+format+"\n", args...)
}

type runReport struct {
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	DurationSeconds float64   `json:"duration_seconds"`
	Format          string    `json:"format"`
	Output          string    `json:"output"`
	EmittedTraces   uint64    `json:"emitted_traces"`
	EmittedSpans    uint64    `json:"emitted_spans"`
	TracesPerSecond float64   `json:"traces_per_second"`
	SpansPerSecond  float64   `json:"spans_per_second"`
}

func Run(cfg config.Config, out io.Writer) error {
	stats := newEmitterStats()
	runStarted := time.Now().UTC()
	debugf(cfg, "starting run format=%s output=%s rate=%.2f/%s duration=%s count=%d workers=%d", cfg.Format, cfg.Output, cfg.RateValue, cfg.RateUnit, cfg.Duration, cfg.Count, cfg.Workers)

	writer := out
	if cfg.Output == "file" {
		if cfg.File == "" {
			return fmt.Errorf("--file is required when --output=file")
		}
		f, err := os.Create(cfg.File)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	}

	buf := bufio.NewWriter(writer)
	defer buf.Flush()

	traceCh := make(chan model.Trace, cfg.BatchSize)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var sinkWG sync.WaitGroup
	var adminWG sync.WaitGroup

	if cfg.HTTPListen != "" {
		adminWG.Add(1)
		go func() {
			defer adminWG.Done()
			if err := runAdminServer(ctx, cfg.HTTPListen, stats); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}()
	}

	sinkWG.Add(1)
	go func() {
		defer sinkWG.Done()
		if err := consumeTraces(ctx, buf, cfg, traceCh, stats); err != nil {
			select {
			case errCh <- err:
			default:
			}
			cancel()
		}
	}()

	if err := produceTraces(ctx, cfg, traceCh); err != nil {
		cancel()
		sinkWG.Wait()
		adminWG.Wait()
		return err
	}
	close(traceCh)
	sinkWG.Wait()
	cancel()
	adminWG.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		finishedAt := time.Now().UTC()
		snapshot := stats.snapshot()
		report := buildRunReport(runStarted, finishedAt, cfg, snapshot)
		if cfg.Output == "noop" {
			if _, err := fmt.Fprintf(out,
				"benchmark summary: traces=%d spans=%d duration=%.2fs traces/sec=%.2f spans/sec=%.2f\n",
				report.EmittedTraces,
				report.EmittedSpans,
				report.DurationSeconds,
				report.TracesPerSecond,
				report.SpansPerSecond,
			); err != nil {
				return err
			}
		}
		if cfg.ReportFile != "" {
			if err := writeRunReport(cfg.ReportFile, report); err != nil {
				return err
			}
		}
		return nil
	}
}

func buildRunReport(startedAt, finishedAt time.Time, cfg config.Config, snapshot statsSnapshot) runReport {
	duration := finishedAt.Sub(startedAt).Seconds()
	if duration <= 0 {
		duration = 1e-9
	}
	return runReport{
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		DurationSeconds: duration,
		Format:          cfg.Format,
		Output:          cfg.Output,
		EmittedTraces:   snapshot.EmittedTraces,
		EmittedSpans:    snapshot.EmittedSpans,
		TracesPerSecond: float64(snapshot.EmittedTraces) / duration,
		SpansPerSecond:  float64(snapshot.EmittedSpans) / duration,
	}
}

func writeRunReport(path string, report runReport) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create report dir: %w", err)
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write report file: %w", err)
	}
	return nil
}

func produceTraces(ctx context.Context, cfg config.Config, traceCh chan<- model.Trace) error {
	jobs := make(chan time.Time, cfg.Workers*2)
	var workersWG sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		workersWG.Add(1)
		go func(workerID int) {
			defer workersWG.Done()
			g := generator.New(withSeed(cfg, cfg.Seed+int64(workerID)))
			for start := range jobs {
				trace := g.GenerateTrace(start)
				select {
				case traceCh <- trace:
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	targetTracesPerInterval := effectiveTracesPerInterval(cfg)
	ratePerSecond := targetTracesPerInterval / cfg.RateInterval.Seconds()
	if ratePerSecond <= 0 {
		close(jobs)
		workersWG.Wait()
		return nil
	}
	tickInterval := 10 * time.Millisecond
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	capacity := ratePerSecond
	if capacity < 1 {
		capacity = 1
	}
	tokens := capacity
	lastRefill := time.Now()
	start := lastRefill.UTC()
	hasDurationLimit := cfg.Count <= 0 && cfg.Duration > 0
	var durationDeadline time.Time
	if hasDurationLimit {
		durationDeadline = start.Add(cfg.Duration)
	}

	sent := 0
	for {
		if cfg.Count > 0 && sent >= cfg.Count {
			break
		}
		if hasDurationLimit && time.Now().After(durationDeadline) {
			break
		}

		now := time.Now()
		elapsed := now.Sub(lastRefill).Seconds()
		lastRefill = now
		tokens += elapsed * ratePerSecond
		if tokens > capacity {
			tokens = capacity
		}

		dispatched := false
		for tokens >= 1 {
			if cfg.Count > 0 && sent >= cfg.Count {
				break
			}
			if hasDurationLimit && time.Now().After(durationDeadline) {
				break
			}

			scheduled := start.Add(time.Duration(float64(sent) / ratePerSecond * float64(time.Second)))
			select {
			case jobs <- scheduled:
				sent++
				tokens -= 1
				dispatched = true
			case <-ctx.Done():
				close(jobs)
				workersWG.Wait()
				return nil
			default:
				tokens = 0
				break
			}
		}

		if dispatched {
			continue
		}
		select {
		case <-ctx.Done():
			close(jobs)
			workersWG.Wait()
			return nil
		case <-ticker.C:
		}
	}

	close(jobs)
	workersWG.Wait()
	return nil
}

func effectiveTracesPerInterval(cfg config.Config) float64 {
	if cfg.RateUnit == config.RateUnitTraces {
		return cfg.RateValue
	}
	estimatedSpansPerTrace := estimateSpansPerTrace(cfg.Depth, cfg.Fanout)
	if estimatedSpansPerTrace <= 0 {
		return 1
	}
	traces := cfg.RateValue / estimatedSpansPerTrace
	if traces < 1 {
		return 1
	}
	return traces
}

func estimateSpansPerTrace(depth int, fanout float64) float64 {
	if depth <= 1 {
		return 1
	}
	if fanout <= 1 {
		return float64(depth)
	}
	total := 1.0
	level := 1.0
	for i := 1; i < depth; i++ {
		level *= fanout
		total += level
	}
	return total
}

func consumeTraces(ctx context.Context, out *bufio.Writer, cfg config.Config, traceCh <-chan model.Trace, stats *emitterStats) error {
	flushTicker := time.NewTicker(cfg.FlushInterval)
	defer flushTicker.Stop()

	var otlpHTTPClient *otlphttp.Client
	var otlpGRPCClient *otlpgrpc.Client
	var zipkinClient *zipkin.Client
	if cfg.Format == "otlp-http" {
		otlpHTTPClient = otlphttp.New(cfg.OTLPEndpoint, cfg.Headers, cfg.Compress == "gzip", cfg.SinkTimeout)
	}
	if cfg.Format == "otlp-grpc" {
		otlpGRPCClient = otlpgrpc.New(cfg.OTLPEndpoint, cfg.Headers, cfg.OTLPInsecure, cfg.SinkTimeout)
		defer otlpGRPCClient.Close()
	}
	if cfg.Format == "zipkin-json" {
		zipkinClient = zipkin.New(cfg.ZipkinEndpoint, cfg.Headers, cfg.SinkTimeout)
	}

	var spanBatch []model.Span
	pendingTraceCount := 0

	networkSem := make(chan struct{}, cfg.SinkMaxInFlight)
	var networkWG sync.WaitGroup
	networkErr := make(chan error, 1)

	reportNetworkErr := func(err error) {
		if err == nil {
			return
		}
		select {
		case networkErr <- err:
		default:
		}
	}

	waitNetwork := func() error {
		networkWG.Wait()
		select {
		case err := <-networkErr:
			return err
		default:
			return nil
		}
	}

	dispatchNetwork := func(send func(context.Context) error, batchTraces, batchSpans int) error {
		select {
		case networkSem <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		networkWG.Add(1)
		go func() {
			defer networkWG.Done()
			defer func() { <-networkSem }()
			debugf(cfg, "sending batch output=%s format=%s traces=%d spans=%d", cfg.Output, cfg.Format, batchTraces, batchSpans)
			if err := sendWithRetry(ctx, cfg.SinkRetries, cfg.SinkRetryBackoff, cfg.SinkTimeout, send); err != nil {
				debugf(cfg, "send failed output=%s format=%s traces=%d spans=%d err=%v", cfg.Output, cfg.Format, batchTraces, batchSpans, err)
				reportNetworkErr(err)
				return
			}
			stats.add(batchTraces, batchSpans)
			debugf(cfg, "send complete output=%s format=%s traces=%d spans=%d", cfg.Output, cfg.Format, batchTraces, batchSpans)
		}()
		return nil
	}

	flushJSONL := func() error {
		if len(spanBatch) == 0 {
			return nil
		}
		batchSpans := len(spanBatch)
		batchTraces := pendingTraceCount
		if err := jsonlenc.WriteTrace(out, model.Trace{Spans: spanBatch}); err != nil {
			return err
		}
		spanBatch = spanBatch[:0]
		pendingTraceCount = 0
		if err := out.Flush(); err != nil {
			return err
		}
		stats.add(batchTraces, batchSpans)
		debugf(cfg, "wrote batch output=%s format=%s traces=%d spans=%d", cfg.Output, cfg.Format, batchTraces, batchSpans)
		return nil
	}
	flushOTLP := func() error {
		if len(spanBatch) == 0 {
			return nil
		}
		batch := append([]model.Span(nil), spanBatch...)
		batchSpans := len(batch)
		batchTraces := pendingTraceCount
		spanBatch = spanBatch[:0]
		pendingTraceCount = 0
		return dispatchNetwork(func(reqCtx context.Context) error {
			return otlpHTTPClient.SendSpans(reqCtx, batch)
		}, batchTraces, batchSpans)
	}
	flushOTLPGRPC := func() error {
		if len(spanBatch) == 0 {
			return nil
		}
		batch := append([]model.Span(nil), spanBatch...)
		batchSpans := len(batch)
		batchTraces := pendingTraceCount
		spanBatch = spanBatch[:0]
		pendingTraceCount = 0
		return dispatchNetwork(func(reqCtx context.Context) error {
			return otlpGRPCClient.SendSpans(reqCtx, batch)
		}, batchTraces, batchSpans)
	}
	flushZipkin := func() error {
		if len(spanBatch) == 0 {
			return nil
		}
		batch := append([]model.Span(nil), spanBatch...)
		batchSpans := len(batch)
		batchTraces := pendingTraceCount
		spanBatch = spanBatch[:0]
		pendingTraceCount = 0
		return dispatchNetwork(func(reqCtx context.Context) error {
			return zipkinClient.SendSpans(reqCtx, batch)
		}, batchTraces, batchSpans)
	}

	finalize := func() error {
		if cfg.Output == "noop" {
			return waitNetwork()
		}
		switch cfg.Format {
		case "jsonl":
			if err := flushJSONL(); err != nil {
				return err
			}
		case "otlp-http":
			if err := flushOTLP(); err != nil {
				return err
			}
		case "otlp-grpc":
			if err := flushOTLPGRPC(); err != nil {
				return err
			}
		case "zipkin-json":
			if err := flushZipkin(); err != nil {
				return err
			}
		default:
			if err := out.Flush(); err != nil {
				return err
			}
		}
		return waitNetwork()
	}

	for {
		select {
		case err := <-networkErr:
			return err
		case <-ctx.Done():
			return finalize()
		case trace, ok := <-traceCh:
			if !ok {
				return finalize()
			}
			if cfg.Output == "noop" {
				stats.add(1, len(trace.Spans))
				continue
			}
			switch cfg.Format {
			case "jsonl":
				spanBatch = append(spanBatch, trace.Spans...)
				pendingTraceCount++
				if len(spanBatch) >= cfg.BatchSize {
					if err := flushJSONL(); err != nil {
						return err
					}
				}
			case "otlp-http":
				spanBatch = append(spanBatch, trace.Spans...)
				pendingTraceCount++
				if len(spanBatch) >= cfg.BatchSize {
					if err := flushOTLP(); err != nil {
						return err
					}
				}
			case "otlp-grpc":
				spanBatch = append(spanBatch, trace.Spans...)
				pendingTraceCount++
				if len(spanBatch) >= cfg.BatchSize {
					if err := flushOTLPGRPC(); err != nil {
						return err
					}
				}
			case "zipkin-json":
				spanBatch = append(spanBatch, trace.Spans...)
				pendingTraceCount++
				if len(spanBatch) >= cfg.BatchSize {
					if err := flushZipkin(); err != nil {
						return err
					}
				}
			case "pretty":
				if _, err := out.WriteString(prettyenc.RenderTrace(trace)); err != nil {
					return err
				}
				if err := out.Flush(); err != nil {
					return err
				}
				stats.add(1, len(trace.Spans))
				debugf(cfg, "wrote trace output=%s format=%s traces=1 spans=%d", cfg.Output, cfg.Format, len(trace.Spans))
			default:
				return fmt.Errorf("unsupported format %q in this stage", cfg.Format)
			}
		case <-flushTicker.C:
			switch cfg.Format {
			case "jsonl":
				if err := flushJSONL(); err != nil {
					return err
				}
			case "otlp-http":
				if err := flushOTLP(); err != nil {
					return err
				}
			case "otlp-grpc":
				if err := flushOTLPGRPC(); err != nil {
					return err
				}
			case "zipkin-json":
				if err := flushZipkin(); err != nil {
					return err
				}
			}
		}
	}
}

func sendWithRetry(ctx context.Context, retries int, backoff, timeout time.Duration, send func(context.Context) error) error {
	if retries < 0 {
		retries = 0
	}
	if backoff <= 0 {
		backoff = 100 * time.Millisecond
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		err := send(reqCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == retries {
			break
		}
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return lastErr
}

func withSeed(cfg config.Config, seed int64) config.Config {
	cfg.Seed = seed
	return cfg
}
