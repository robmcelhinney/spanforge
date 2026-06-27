package generator

import (
	"strings"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func (g *Generator) applyModes(trace *model.Trace) {
	if trace == nil || len(trace.Spans) == 0 {
		return
	}
	g.applyWeirdModes(trace)
	g.applyInvalidModes(trace)
}

func (g *Generator) applyWeirdModes(trace *model.Trace) {
	for _, mode := range g.cfg.Weird {
		switch mode {
		case "future-timestamp":
			shiftTrace(trace, 24*time.Hour)
			markTrace(trace, "future-timestamp")
		case "clock-skew":
			applyClockSkew(trace)
			markTrace(trace, "clock-skew")
		case "huge-duration":
			applyHugeDuration(trace)
			markTrace(trace, "huge-duration")
		case "high-cardinality-route":
			applyHighCardinalityRoute(trace)
			markTrace(trace, "high-cardinality-route")
		case "huge-attribute":
			applyHugeAttribute(trace)
			markTrace(trace, "huge-attribute")
		case "mixed-semconv":
			applyMixedSemConv(trace)
			markTrace(trace, "mixed-semconv")
		}
	}
}

func (g *Generator) applyInvalidModes(trace *model.Trace) {
	for _, mode := range g.cfg.Invalid {
		switch mode {
		case "duplicate-span-id":
			if len(trace.Spans) > 1 {
				trace.Spans[1].SpanID = trace.Spans[0].SpanID
			}
			markTrace(trace, "duplicate-span-id")
		case "negative-duration":
			trace.Spans[len(trace.Spans)-1].Duration = -1 * time.Millisecond
			markTrace(trace, "negative-duration")
		case "empty-required-fields":
			applyEmptyRequiredFields(trace)
			markTrace(trace, "empty-required-fields")
		}
	}
}

func shiftTrace(trace *model.Trace, delta time.Duration) {
	for i := range trace.Spans {
		trace.Spans[i].StartTime = trace.Spans[i].StartTime.Add(delta)
		for j := range trace.Spans[i].Events {
			trace.Spans[i].Events[j].Time = trace.Spans[i].Events[j].Time.Add(delta)
		}
	}
}

func applyClockSkew(trace *model.Trace) {
	for i := range trace.Spans {
		if trace.Spans[i].HasParent {
			if i%2 == 0 {
				trace.Spans[i].StartTime = trace.Spans[i].StartTime.Add(-7 * time.Millisecond)
			} else {
				trace.Spans[i].StartTime = trace.Spans[i].StartTime.Add(4 * time.Millisecond)
			}
			if trace.Spans[i].Attributes == nil {
				trace.Spans[i].Attributes = model.Attrs{}
			}
			trace.Spans[i].Attributes["clock.skew_ms"] = int64(trace.Spans[i].StartTime.Sub(trace.Spans[0].StartTime) / time.Millisecond)
		}
	}
}

func applyHugeDuration(trace *model.Trace) {
	for i := range trace.Spans {
		if i == 0 {
			trace.Spans[i].Duration *= 50
			if trace.Spans[i].Attributes == nil {
				trace.Spans[i].Attributes = model.Attrs{}
			}
			trace.Spans[i].Attributes["spanforge.weird"] = "huge-duration"
			return
		}
	}
}

func applyHighCardinalityRoute(trace *model.Trace) {
	for i := range trace.Spans {
		if route, ok := trace.Spans[i].Attributes["http.route"].(string); ok && route != "" {
			trace.Spans[i].Attributes["http.route"] = route + "?request=" + fmtHexID(trace.Spans[i].TraceID, trace.Spans[i].SpanID)
		}
		if route, ok := trace.Spans[i].Attributes["rpc.method"].(string); ok && route != "" {
			trace.Spans[i].Attributes["rpc.method"] = route + "-" + fmtHexID(trace.Spans[i].TraceID, trace.Spans[i].SpanID)
		}
	}
}

func applyHugeAttribute(trace *model.Trace) {
	blob := strings.Repeat("spanforge-", 1024)
	for i := range trace.Spans {
		if trace.Spans[i].Attributes == nil {
			trace.Spans[i].Attributes = model.Attrs{}
		}
		trace.Spans[i].Attributes["spanforge.blob"] = blob
		if i == 0 {
			if trace.Spans[i].Resource.Attributes == nil {
				trace.Spans[i].Resource.Attributes = model.Attrs{}
			}
			trace.Spans[i].Resource.Attributes["spanforge.blob"] = blob
		}
	}
}

func applyMixedSemConv(trace *model.Trace) {
	for i := range trace.Spans {
		if trace.Spans[i].Attributes == nil {
			trace.Spans[i].Attributes = model.Attrs{}
		}
		if route, ok := trace.Spans[i].Attributes["http.route"].(string); ok && route != "" {
			trace.Spans[i].Attributes["url.path"] = route
			trace.Spans[i].Attributes["http.target"] = route
		}
		if method, ok := trace.Spans[i].Attributes["http.method"].(string); ok && method != "" {
			trace.Spans[i].Attributes["http.request.method"] = method
		}
		if svc, ok := trace.Spans[i].Attributes["service.name"].(string); ok && svc != "" {
			trace.Spans[i].Attributes["server.address"] = svc
		}
	}
}

func applyEmptyRequiredFields(trace *model.Trace) {
	if len(trace.Spans) == 0 {
		return
	}
	span := &trace.Spans[0]
	span.Name = ""
	span.Kind = ""
	span.StartTime = time.Time{}
	if span.Attributes == nil {
		span.Attributes = model.Attrs{}
	}
	span.Attributes["service.name"] = ""
	if span.Resource.Attributes == nil {
		span.Resource.Attributes = model.Attrs{}
	}
	span.Resource.Attributes["service.name"] = ""
}

func markTrace(trace *model.Trace, mode string) {
	if len(trace.Spans) == 0 {
		return
	}
	if trace.Spans[0].Attributes == nil {
		trace.Spans[0].Attributes = model.Attrs{}
	}
	trace.Spans[0].Attributes["spanforge.weird"] = mode
}
