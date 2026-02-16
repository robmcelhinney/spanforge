package jsonl

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

type spanLine struct {
	TraceID      string         `json:"trace_id"`
	SpanID       string         `json:"span_id"`
	ParentSpanID string         `json:"parent_id,omitempty"`
	Name         string         `json:"name"`
	Kind         string         `json:"kind"`
	ServiceName  string         `json:"service_name,omitempty"`
	StartTime    time.Time      `json:"start_time"`
	DurationMS   float64        `json:"duration_ms"`
	Status       string         `json:"status"`
	Attributes   map[string]any `json:"attributes,omitempty"`
}

func WriteTrace(w io.Writer, trace model.Trace) error {
	enc := json.NewEncoder(w)
	for _, span := range trace.Spans {
		line := spanLine{
			TraceID:    hex.EncodeToString(span.TraceID[:]),
			SpanID:     hex.EncodeToString(span.SpanID[:]),
			Name:       span.Name,
			Kind:       span.Kind,
			StartTime:  span.StartTime.UTC(),
			DurationMS: float64(span.Duration) / float64(time.Millisecond),
			Status:     span.Status.Code,
			Attributes: span.Attributes,
		}
		if span.HasParent {
			line.ParentSpanID = hex.EncodeToString(span.ParentSpanID[:])
		}
		if v, ok := span.Attributes["service.name"].(string); ok {
			line.ServiceName = v
		}
		if err := enc.Encode(line); err != nil {
			return fmt.Errorf("encode jsonl line: %w", err)
		}
	}
	return nil
}
