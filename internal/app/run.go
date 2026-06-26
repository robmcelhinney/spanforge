package app

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
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
	"gopkg.in/yaml.v3"
)

func debugf(cfg config.Config, format string, args ...any) {
	if !cfg.Debug {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "spanforge debug: "+format+"\n", args...)
}

type runReport struct {
	StartedAt       time.Time     `json:"started_at"`
	FinishedAt      time.Time     `json:"finished_at"`
	DurationSeconds float64       `json:"duration_seconds"`
	RunID           string        `json:"run_id"`
	Profile         string        `json:"profile"`
	Format          string        `json:"format"`
	Output          string        `json:"output"`
	EmittedTraces   uint64        `json:"emitted_traces"`
	EmittedSpans    uint64        `json:"emitted_spans"`
	TracesPerSecond float64       `json:"traces_per_second"`
	SpansPerSecond  float64       `json:"spans_per_second"`
	Services        []string      `json:"services"`
	SampleTraceIDs  []string      `json:"sample_trace_ids"`
	Phases          []phaseReport `json:"phases,omitempty"`
}

type phaseReport struct {
	Name       string `json:"name"`
	TracesSent uint64 `json:"traces_sent"`
	SpansSent  uint64 `json:"spans_sent"`
}

type reportManifest struct {
	services       map[string]struct{}
	sampleTraceIDs []string
	seenTraceIDs   map[string]struct{}
	phases         map[string]*phaseReport
	phaseOrder     []string
}

func newReportManifest() *reportManifest {
	return &reportManifest{
		services:     map[string]struct{}{},
		seenTraceIDs: map[string]struct{}{},
		phases:       map[string]*phaseReport{},
	}
}

func (m *reportManifest) observe(trace model.Trace) {
	traceID := fmtTraceID(trace.TraceID)
	if _, ok := m.seenTraceIDs[traceID]; !ok {
		m.seenTraceIDs[traceID] = struct{}{}
		if len(m.sampleTraceIDs) < 20 {
			m.sampleTraceIDs = append(m.sampleTraceIDs, traceID)
		}
	}
	for _, span := range trace.Spans {
		if service, ok := span.Attributes["service.name"].(string); ok && service != "" {
			m.services[service] = struct{}{}
		}
		if service, ok := span.Resource.Attributes["service.name"].(string); ok && service != "" {
			m.services[service] = struct{}{}
		}
	}
	if phase := tracePhase(trace); phase != "" {
		p, ok := m.phases[phase]
		if !ok {
			m.phaseOrder = append(m.phaseOrder, phase)
			p = &phaseReport{Name: phase}
			m.phases[phase] = p
		}
		p.TracesSent++
		p.SpansSent += uint64(len(trace.Spans))
	}
}

func tracePhase(trace model.Trace) string {
	for _, span := range trace.Spans {
		if phase, ok := span.Attributes["spanforge.phase"].(string); ok && phase != "" {
			return phase
		}
		if phase, ok := span.Resource.Attributes["spanforge.phase"].(string); ok && phase != "" {
			return phase
		}
	}
	return ""
}

func (m *reportManifest) snapshot() reportManifestSnapshot {
	services := make([]string, 0, len(m.services))
	for service := range m.services {
		services = append(services, service)
	}
	sort.Strings(services)
	phases := make([]phaseReport, 0, len(m.phaseOrder))
	for _, name := range m.phaseOrder {
		phases = append(phases, *m.phases[name])
	}
	return reportManifestSnapshot{
		Services:       services,
		SampleTraceIDs: append([]string(nil), m.sampleTraceIDs...),
		Phases:         phases,
	}
}

type reportManifestSnapshot struct {
	Services       []string
	SampleTraceIDs []string
	Phases         []phaseReport
}

func Run(cfg config.Config, out io.Writer) error {
	cfg.RunID = effectiveRunID(cfg)
	stats := newEmitterStats()
	manifest := newReportManifest()
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
		if err := consumeTraces(ctx, buf, cfg, traceCh, stats, manifest); err != nil {
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
		report := buildRunReport(runStarted, finishedAt, cfg, snapshot, manifest.snapshot())
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

func buildRunReport(startedAt, finishedAt time.Time, cfg config.Config, snapshot statsSnapshot, manifest reportManifestSnapshot) runReport {
	duration := finishedAt.Sub(startedAt).Seconds()
	if duration <= 0 {
		duration = 1e-9
	}
	return runReport{
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		DurationSeconds: duration,
		RunID:           cfg.RunID,
		Profile:         cfg.Profile,
		Format:          cfg.Format,
		Output:          cfg.Output,
		EmittedTraces:   snapshot.EmittedTraces,
		EmittedSpans:    snapshot.EmittedSpans,
		TracesPerSecond: float64(snapshot.EmittedTraces) / duration,
		SpansPerSecond:  float64(snapshot.EmittedSpans) / duration,
		Services:        manifest.Services,
		SampleTraceIDs:  manifest.SampleTraceIDs,
		Phases:          manifest.Phases,
	}
}

func effectiveRunID(cfg config.Config) string {
	if cfg.RunID != "" {
		return cfg.RunID
	}
	return fmt.Sprintf("sf_seed_%d", cfg.Seed)
}

func fmtTraceID(id model.TraceID) string {
	return hex.EncodeToString(id[:])
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
	phases, err := loadPhases(cfg)
	if err != nil {
		return err
	}
	if len(phases) > 0 {
		return produceTracePhases(ctx, cfg, phases, traceCh)
	}
	return produceTraceSteady(ctx, cfg, traceCh)
}

func produceTracePhases(ctx context.Context, cfg config.Config, phases []loadPhase, traceCh chan<- model.Trace) error {
	totalDuration := time.Duration(0)
	for _, phase := range phases {
		totalDuration += phase.Duration
	}
	remainingCount := cfg.Count
	for i, phase := range phases {
		phaseCfg := phase.apply(cfg)
		phaseCfg.Seed = cfg.Seed + int64(i*maxInt(1, cfg.Workers))
		if err := phaseCfg.Validate(); err != nil {
			return fmt.Errorf("invalid phase %q: %w", phase.Name, err)
		}
		if cfg.Count > 0 {
			phaseCfg.Count = phaseCount(cfg.Count, remainingCount, phase.Duration, totalDuration, i == len(phases)-1)
			remainingCount -= phaseCfg.Count
			if phaseCfg.Count <= 0 {
				continue
			}
		}
		debugf(phaseCfg, "starting phase name=%s rate=%.2f/%s duration=%s count=%d errors=%.4f retries=%.4f p95=%s", phase.Name, phaseCfg.RateValue, phaseCfg.RateUnit, phaseCfg.Duration, phaseCfg.Count, phaseCfg.Errors, phaseCfg.Retries, phaseCfg.P95)
		if err := produceTraceSteady(ctx, phaseCfg, traceCh); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
	return nil
}

func produceTraceSteady(ctx context.Context, cfg config.Config, traceCh chan<- model.Trace) error {
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

type loadPhase struct {
	Name     string
	Duration time.Duration
	Rate     *float64
	RateUnit *config.RateUnit
	Errors   *float64
	Retries  *float64
	P50      *time.Duration
	P95      *time.Duration
	P99      *time.Duration
}

func (p loadPhase) apply(cfg config.Config) config.Config {
	cfg.Phase = p.Name
	cfg.Duration = p.Duration
	if p.Rate != nil {
		cfg.RateValue = *p.Rate
	}
	if p.RateUnit != nil {
		cfg.RateUnit = *p.RateUnit
	}
	if p.Errors != nil {
		cfg.Errors = *p.Errors
	}
	if p.Retries != nil {
		cfg.Retries = *p.Retries
	}
	if p.P50 != nil {
		cfg.P50 = *p.P50
	}
	if p.P95 != nil {
		cfg.P95 = *p.P95
	}
	if p.P99 != nil {
		cfg.P99 = *p.P99
	}
	return cfg
}

type phaseFile struct {
	Phases []phaseFileItem `yaml:"phases"`
}

type phaseFileItem struct {
	Name     string   `yaml:"name"`
	Duration string   `yaml:"duration"`
	Rate     *float64 `yaml:"rate"`
	RateUnit *string  `yaml:"rate_unit"`
	Errors   *string  `yaml:"errors"`
	Retries  *string  `yaml:"retries"`
	P50      *string  `yaml:"p50"`
	P95      *string  `yaml:"p95"`
	P99      *string  `yaml:"p99"`
}

func loadPhases(cfg config.Config) ([]loadPhase, error) {
	if cfg.PhaseFile != "" {
		return loadPhaseFile(cfg.PhaseFile)
	}
	if cfg.Load != "" {
		return builtInLoad(cfg)
	}
	return nil, nil
}

func loadPhaseFile(path string) ([]loadPhase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read phase file: %w", err)
	}
	var file phaseFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse phase file: %w", err)
	}
	if len(file.Phases) == 0 {
		return nil, fmt.Errorf("phase file must contain at least one phase")
	}
	phases := make([]loadPhase, 0, len(file.Phases))
	for _, item := range file.Phases {
		phase, err := parsePhaseFileItem(item)
		if err != nil {
			return nil, err
		}
		phases = append(phases, phase)
	}
	return phases, nil
}

func parsePhaseFileItem(item phaseFileItem) (loadPhase, error) {
	if item.Name == "" {
		return loadPhase{}, fmt.Errorf("phase name is required")
	}
	duration, err := time.ParseDuration(item.Duration)
	if err != nil || duration <= 0 {
		return loadPhase{}, fmt.Errorf("invalid duration for phase %q", item.Name)
	}
	phase := loadPhase{Name: item.Name, Duration: duration, Rate: item.Rate}
	if item.RateUnit != nil {
		unit, err := config.ParseRateUnit(*item.RateUnit)
		if err != nil {
			return loadPhase{}, fmt.Errorf("invalid rate_unit for phase %q: %w", item.Name, err)
		}
		phase.RateUnit = &unit
	}
	var parseErr error
	phase.Errors, parseErr = optionalPercent(item.Errors, "errors", item.Name)
	if parseErr != nil {
		return loadPhase{}, parseErr
	}
	phase.Retries, parseErr = optionalPercent(item.Retries, "retries", item.Name)
	if parseErr != nil {
		return loadPhase{}, parseErr
	}
	phase.P50, parseErr = optionalDuration(item.P50, "p50", item.Name)
	if parseErr != nil {
		return loadPhase{}, parseErr
	}
	phase.P95, parseErr = optionalDuration(item.P95, "p95", item.Name)
	if parseErr != nil {
		return loadPhase{}, parseErr
	}
	phase.P99, parseErr = optionalDuration(item.P99, "p99", item.Name)
	if parseErr != nil {
		return loadPhase{}, parseErr
	}
	return phase, nil
}

func optionalPercent(raw *string, field, phase string) (*float64, error) {
	if raw == nil {
		return nil, nil
	}
	v, err := config.ParsePercent(*raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s for phase %q: %w", field, phase, err)
	}
	return &v, nil
}

func optionalDuration(raw *string, field, phase string) (*time.Duration, error) {
	if raw == nil {
		return nil, nil
	}
	d, err := time.ParseDuration(*raw)
	if err != nil || d <= 0 {
		return nil, fmt.Errorf("invalid %s for phase %q", field, phase)
	}
	return &d, nil
}

func builtInLoad(cfg config.Config) ([]loadPhase, error) {
	total := cfg.Duration
	if total <= 0 {
		total = 30 * time.Second
	}
	rate := func(v float64) *float64 { return &v }
	percent := func(v float64) *float64 { return &v }
	dur := func(v time.Duration) *time.Duration { return &v }
	switch cfg.Load {
	case "steady":
		return []loadPhase{{Name: "steady", Duration: total}}, nil
	case "warmup-spike-recovery":
		return []loadPhase{
			{Name: "warmup", Duration: fractionDuration(total, 4), Rate: rate(cfg.RateValue * 0.5)},
			{Name: "spike", Duration: fractionDuration(total, 4), Rate: rate(cfg.RateValue * 4), Errors: percent(maxFloat(cfg.Errors, 0.02))},
			{Name: "recovery", Duration: total - 2*fractionDuration(total, 4), Rate: rate(cfg.RateValue)},
		}, nil
	case "brownout":
		return []loadPhase{
			{Name: "baseline", Duration: fractionDuration(total, 4)},
			{Name: "brownout", Duration: total - fractionDuration(total, 4), Errors: percent(maxFloat(cfg.Errors, 0.15)), Retries: percent(maxFloat(cfg.Retries, 0.08)), P95: dur(maxDuration(cfg.P95, 2*time.Second)), P99: dur(maxDuration(cfg.P99, 4*time.Second))},
		}, nil
	case "sawtooth":
		q := fractionDuration(total, 4)
		return []loadPhase{
			{Name: "low-1", Duration: q, Rate: rate(cfg.RateValue * 0.5)},
			{Name: "high-1", Duration: q, Rate: rate(cfg.RateValue * 2)},
			{Name: "low-2", Duration: q, Rate: rate(cfg.RateValue * 0.5)},
			{Name: "high-2", Duration: total - 3*q, Rate: rate(cfg.RateValue * 2)},
		}, nil
	case "error-storm":
		return []loadPhase{
			{Name: "baseline", Duration: fractionDuration(total, 3)},
			{Name: "error-storm", Duration: total - fractionDuration(total, 3), Errors: percent(maxFloat(cfg.Errors, 0.25)), Retries: percent(maxFloat(cfg.Retries, 0.12))},
		}, nil
	default:
		return nil, fmt.Errorf("unknown load preset %q", cfg.Load)
	}
}

func phaseCount(totalCount, remainingCount int, phaseDuration, totalDuration time.Duration, last bool) int {
	if last {
		return remainingCount
	}
	if totalDuration <= 0 {
		return 0
	}
	count := int(math.Ceil(float64(totalCount) * float64(phaseDuration) / float64(totalDuration)))
	if count > remainingCount {
		return remainingCount
	}
	return count
}

func fractionDuration(total time.Duration, parts int) time.Duration {
	if parts <= 0 {
		return total
	}
	d := total / time.Duration(parts)
	if d <= 0 {
		return time.Nanosecond
	}
	return d
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
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

func consumeTraces(ctx context.Context, out *bufio.Writer, cfg config.Config, traceCh <-chan model.Trace, stats *emitterStats, manifest *reportManifest) error {
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
			manifest.observe(trace)
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
