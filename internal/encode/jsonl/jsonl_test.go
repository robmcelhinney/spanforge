package jsonl

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func TestWriteTrace(t *testing.T) {
	trace := model.Trace{}
	trace.Spans = []model.Span{{
		Name:       "root",
		Kind:       "SERVER",
		StartTime:  time.Unix(1, 0).UTC(),
		Duration:   25 * time.Millisecond,
		Status:     model.SpanStatus{Code: "OK"},
		Attributes: model.Attrs{"service.name": "api"},
	}}

	var buf bytes.Buffer
	if err := WriteTrace(&buf, trace); err != nil {
		t.Fatalf("WriteTrace: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\"trace_id\"") || !strings.Contains(out, "\"span_id\"") {
		t.Fatalf("missing ids in output: %s", out)
	}
	if !strings.Contains(out, "\"name\":\"root\"") {
		t.Fatalf("missing span name in output: %s", out)
	}
}
