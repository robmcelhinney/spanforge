package generator

import (
	"encoding/hex"
	"math"
	"strings"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
	"github.com/robmcelhinney/spanforge/internal/model"
)

const z95 = 1.6448536269514722

type Generator struct {
	cfg      config.Config
	rng      *RNG
	topology Topology
	profile  profileModule
	mu       float64
	sigma    float64
}

func New(cfg config.Config) *Generator {
	mu := math.Log(float64(cfg.P50))
	sigma := (math.Log(float64(cfg.P95)) - mu) / z95
	if sigma < 0.01 {
		sigma = 0.01
	}

	return &Generator{
		cfg:      cfg,
		rng:      NewRNG(cfg.Seed),
		topology: BuildTopology(cfg.ServicePrefix, cfg.Services),
		profile:  moduleFor(cfg.Profile),
		mu:       mu,
		sigma:    sigma,
	}
}

func (g *Generator) GenerateTrace(start time.Time) model.Trace {
	traceID := g.newTraceID()
	trace := model.Trace{
		TraceID: traceID,
		Resource: model.Resource{Attributes: model.Attrs{
			"deployment.environment": "dev",
		}},
	}

	rootID := g.newSpanID()
	routeIdx := g.rng.Intn(max(1, g.cfg.Routes))
	root := g.profile.buildRoot(g.topology.Frontdoor, routeIdx, start, rootID, traceID, g.sampleDurationForProfile())
	g.applyCardinalityAttrs(&root)
	g.maybeAddProfileEvent(&root)
	retrySpan := g.maybeErrorAndRetry(&root)
	trace.Spans = append(trace.Spans, root)
	if retrySpan != nil {
		g.applyCardinalityAttrs(retrySpan)
		trace.Spans = append(trace.Spans, *retrySpan)
	}

	g.generateChildren(&trace, root, 1)
	return trace
}

func (g *Generator) generateChildren(trace *model.Trace, parent model.Span, level int) {
	if level >= g.cfg.Depth {
		return
	}

	childCount := int(g.cfg.Fanout)
	if g.rng.Float64() < g.cfg.Fanout-float64(childCount) {
		childCount++
	}
	if childCount < 1 {
		childCount = 1
	}

	for i := 0; i < childCount; i++ {
		service := g.topology.Services[g.rng.Intn(len(g.topology.Services))]
		start := parent.StartTime.Add(time.Duration(i+1) * time.Millisecond)
		routeIdx := g.rng.Intn(max(1, g.cfg.Routes))
		if g.cfg.Profile == "queue" {
			producer, consumer := g.profile.buildQueuePair(parent, service, routeIdx, start, g.newSpanID(), g.newSpanID(), trace.TraceID, g.sampleDurationForProfile())
			g.applyCardinalityAttrs(&producer)
			g.applyCardinalityAttrs(&consumer)
			g.maybeAddProfileEvent(&producer)
			g.maybeAddProfileEvent(&consumer)
			producerRetry := g.maybeErrorAndRetry(&producer)
			consumerRetry := g.maybeErrorAndRetry(&consumer)
			trace.Spans = append(trace.Spans, producer, consumer)
			if producerRetry != nil {
				g.applyCardinalityAttrs(producerRetry)
				trace.Spans = append(trace.Spans, *producerRetry)
			}
			if consumerRetry != nil {
				g.applyCardinalityAttrs(consumerRetry)
				trace.Spans = append(trace.Spans, *consumerRetry)
			}
			g.generateChildren(trace, consumer, level+1)
			continue
		}
		child := g.profile.buildChild(parent, service, routeIdx, start, g.newSpanID(), trace.TraceID, g.sampleDurationForProfile(), g.rng.Float64() < g.cfg.DBHeavy, g.rng.Float64() < g.cfg.CacheHitRate)
		g.applyCardinalityAttrs(&child)
		g.maybeAddProfileEvent(&child)
		retrySpan := g.maybeErrorAndRetry(&child)
		trace.Spans = append(trace.Spans, child)
		if retrySpan != nil {
			g.applyCardinalityAttrs(retrySpan)
			trace.Spans = append(trace.Spans, *retrySpan)
		}
		g.generateChildren(trace, child, level+1)
	}
}

func (g *Generator) maybeAddProfileEvent(span *model.Span) {
	if g.rng.Float64() >= g.eventProbability() {
		return
	}
	attrs := model.Attrs{"profile": g.cfg.Profile}
	switch g.cfg.Profile {
	case "grpc":
		span.Events = append(span.Events, model.Event{Name: "grpc.message", Time: span.StartTime.Add(span.Duration / 3), Attributes: attrs})
	case "queue":
		span.Events = append(span.Events, model.Event{Name: "message.visible", Time: span.StartTime.Add(span.Duration / 3), Attributes: attrs})
	case "batch":
		span.Events = append(span.Events, model.Event{Name: "batch.chunk.complete", Time: span.StartTime.Add(span.Duration / 3), Attributes: attrs})
	default:
		span.Events = append(span.Events, model.Event{Name: "app.log", Time: span.StartTime.Add(span.Duration / 3), Attributes: attrs})
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g *Generator) maybeErrorAndRetry(span *model.Span) *model.Span {
	if g.rng.Float64() >= g.errorProbability(span) {
		return nil
	}
	span.Status = model.SpanStatus{Code: "ERROR", Message: "synthetic failure"}
	span.Attributes["error"] = true
	if _, ok := span.Attributes["http.method"]; ok {
		span.Attributes["http.status_code"] = 500
	}
	span.Events = append(span.Events, model.Event{
		Name: "exception",
		Time: span.StartTime.Add(span.Duration / 2),
		Attributes: model.Attrs{
			"exception.type":    "SyntheticError",
			"exception.message": "generated error",
		},
	})

	if g.rng.Float64() < g.cfg.Retries {
		span.Events = append(span.Events, model.Event{
			Name: "retry",
			Time: span.StartTime.Add(span.Duration),
			Attributes: model.Attrs{
				"retry.attempt": 1,
			},
		})
		retryDuration := g.sampleDuration() / 2
		span.Duration += retryDuration
		serviceName, _ := span.Attributes["service.name"].(string)
		retrySpan := &model.Span{
			TraceID:      span.TraceID,
			SpanID:       g.newSpanID(),
			ParentSpanID: span.SpanID,
			HasParent:    true,
			Name:         "retry attempt",
			Kind:         "INTERNAL",
			StartTime:    span.StartTime.Add(span.Duration - retryDuration),
			Duration:     retryDuration,
			Attributes: model.Attrs{
				"service.name":  serviceName,
				"retry.attempt": 1,
			},
			Status:   model.SpanStatus{Code: "OK"},
			Resource: model.Resource{Attributes: model.Attrs{"service.name": serviceName}},
		}
		if _, ok := span.Attributes["http.method"]; ok {
			retrySpan.Attributes["http.status_code"] = 200
		}
		return retrySpan
	}
	return nil
}

func (g *Generator) eventProbability() float64 {
	switch g.variety() {
	case "low":
		return 0.15
	case "high":
		return 0.65
	default:
		return 0.35
	}
}

func (g *Generator) errorProbability(span *model.Span) float64 {
	rate := g.cfg.Errors * g.profileErrorMultiplier()
	if span.Duration >= g.cfg.P95 {
		rate += g.cfg.Errors*0.75 + 0.02
	}
	if span.Duration >= g.cfg.P99 {
		rate += g.cfg.Errors + 0.03
	}
	switch g.variety() {
	case "low":
		rate -= 0.01
	case "high":
		rate += 0.02
	}
	if rate < 0 {
		return 0
	}
	if rate > 1 {
		return 1
	}
	return rate
}

func (g *Generator) sampleDurationForProfile() time.Duration {
	d := g.sampleDuration()
	d = time.Duration(float64(d) * g.profileLatencyMultiplier())
	if g.rng.Float64() < g.slowProbability() {
		d = time.Duration(float64(d) * g.slowMultiplier())
	}
	if d < time.Microsecond {
		return time.Microsecond
	}
	return d
}

func (g *Generator) profileLatencyMultiplier() float64 {
	switch strings.ToLower(g.cfg.Profile) {
	case "grpc":
		return 1.15
	case "queue":
		return 1.4
	case "batch":
		return 1.8
	default:
		return 1.0
	}
}

func (g *Generator) profileErrorMultiplier() float64 {
	switch strings.ToLower(g.cfg.Profile) {
	case "grpc":
		return 0.9
	case "queue":
		return 1.2
	case "batch":
		return 1.1
	default:
		return 1.0
	}
}

func (g *Generator) slowProbability() float64 {
	switch g.variety() {
	case "low":
		return 0.01
	case "high":
		return 0.15
	default:
		return 0.05
	}
}

func (g *Generator) slowMultiplier() float64 {
	switch g.variety() {
	case "low":
		return 2.0
	case "high":
		return 8.0
	default:
		return 4.0
	}
}

func (g *Generator) variety() string {
	v := strings.ToLower(strings.TrimSpace(g.cfg.Variety))
	if v == "" {
		return "medium"
	}
	return v
}

func (g *Generator) applyCardinalityAttrs(span *model.Span) {
	if !g.cfg.HighCardinality {
		return
	}
	if span.Attributes == nil {
		span.Attributes = model.Attrs{}
	}
	span.Attributes["span.id"] = hex.EncodeToString(span.SpanID[:])
	span.Attributes["trace.id"] = hex.EncodeToString(span.TraceID[:])
	if _, ok := span.Attributes["http.method"]; ok {
		span.Attributes["http.request_id"] = fmtHexID(span.TraceID, span.SpanID)
	}
	if _, ok := span.Attributes["messaging.system"]; ok {
		span.Attributes["messaging.message_id"] = fmtHexID(span.TraceID, span.SpanID)
	}
	if _, ok := span.Attributes["batch.job"]; ok {
		span.Attributes["batch.run_id"] = fmtHexID(span.TraceID, span.SpanID)
	}
}

func fmtHexID(traceID model.TraceID, spanID model.SpanID) string {
	return hex.EncodeToString(traceID[:]) + "-" + hex.EncodeToString(spanID[:])
}

func (g *Generator) sampleDuration() time.Duration {
	u1 := g.rng.Float64()
	if u1 < 1e-9 {
		u1 = 1e-9
	}
	u2 := g.rng.Float64()
	z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	x := math.Exp(g.mu + g.sigma*z)
	if x < float64(time.Microsecond) {
		x = float64(time.Microsecond)
	}
	return time.Duration(x)
}

func (g *Generator) newTraceID() model.TraceID {
	var id model.TraceID
	g.rng.Read(id[:])
	return id
}

func (g *Generator) newSpanID() model.SpanID {
	var id model.SpanID
	g.rng.Read(id[:])
	return id
}
