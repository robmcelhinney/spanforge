package pretty

import (
	"strings"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func TestRenderTrace(t *testing.T) {
	var traceID model.TraceID
	var rootID model.SpanID
	var childID model.SpanID
	rootID[0] = 1
	childID[0] = 2

	trace := model.Trace{TraceID: traceID, Spans: []model.Span{
		{
			TraceID:    traceID,
			SpanID:     rootID,
			Name:       "GET /",
			Kind:       "SERVER",
			StartTime:  time.Unix(1, 0),
			Duration:   10 * time.Millisecond,
			Status:     model.SpanStatus{Code: "OK"},
			Attributes: model.Attrs{"service.name": "api"},
		},
		{
			TraceID:      traceID,
			SpanID:       childID,
			ParentSpanID: rootID,
			HasParent:    true,
			Name:         "call db",
			Kind:         "CLIENT",
			StartTime:    time.Unix(1, 1),
			Duration:     5 * time.Millisecond,
			Status:       model.SpanStatus{Code: "OK"},
			Attributes:   model.Attrs{"service.name": "api"},
		},
	}}

	out := RenderTrace(trace)
	if !strings.Contains(out, "trace ") || !strings.Contains(out, "GET /") || !strings.Contains(out, "call db") {
		t.Fatalf("unexpected render output: %s", out)
	}
}
