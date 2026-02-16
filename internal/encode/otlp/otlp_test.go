package otlp

import (
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func TestEncodeSpans(t *testing.T) {
	var tid model.TraceID
	var sid model.SpanID
	tid[0] = 1
	sid[0] = 2
	spans := []model.Span{{
		TraceID:   tid,
		SpanID:    sid,
		Name:      "root",
		Kind:      "SERVER",
		StartTime: time.Unix(1, 0).UTC(),
		Duration:  10 * time.Millisecond,
		Status:    model.SpanStatus{Code: "OK"},
		Attributes: model.Attrs{
			"service.name": "api",
		},
	}}

	req, err := EncodeSpans(spans)
	if err != nil {
		t.Fatalf("EncodeSpans: %v", err)
	}
	if len(req.ResourceSpans) != 1 {
		t.Fatalf("resource spans = %d, want 1", len(req.ResourceSpans))
	}
	if len(req.ResourceSpans[0].ScopeSpans) != 1 {
		t.Fatalf("scope spans = %d, want 1", len(req.ResourceSpans[0].ScopeSpans))
	}
	if len(req.ResourceSpans[0].ScopeSpans[0].Spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(req.ResourceSpans[0].ScopeSpans[0].Spans))
	}
}
