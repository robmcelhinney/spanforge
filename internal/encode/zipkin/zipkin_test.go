package zipkin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func TestEncodeSpans(t *testing.T) {
	start := time.Unix(1700000000, 0).UTC()
	traceID := model.TraceID{1, 2, 3, 4}
	parentID := model.SpanID{9, 9, 9, 9}
	spanID := model.SpanID{5, 6, 7, 8}
	spans := []model.Span{{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentID,
		HasParent:    true,
		Name:         "GET /route-1",
		Kind:         "CLIENT",
		StartTime:    start,
		Duration:     5 * time.Millisecond,
		Attributes: model.Attrs{
			"service.name": "svc-a",
			"http.method":  "GET",
			"cache.hit":    true,
		},
		Status: model.SpanStatus{Code: "OK"},
		Resource: model.Resource{Attributes: model.Attrs{
			"service.name": "svc-a",
		}},
	}}

	payload, err := EncodeSpans(spans)
	if err != nil {
		t.Fatalf("EncodeSpans: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want=1", len(got))
	}
	if got[0]["traceId"] == "" {
		t.Fatal("missing traceId")
	}
	if got[0]["id"] == "" {
		t.Fatal("missing id")
	}
	if got[0]["parentId"] == "" {
		t.Fatal("missing parentId")
	}
	if got[0]["duration"].(float64) <= 0 {
		t.Fatal("duration must be > 0")
	}
	tags, ok := got[0]["tags"].(map[string]any)
	if !ok {
		t.Fatal("missing tags")
	}
	if tags["http.method"] != "GET" {
		t.Fatalf("http.method=%v", tags["http.method"])
	}
}
