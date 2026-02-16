package zipkin

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
)

type endpoint struct {
	ServiceName string `json:"serviceName,omitempty"`
}

type span struct {
	TraceID       string            `json:"traceId"`
	ID            string            `json:"id"`
	ParentID      string            `json:"parentId,omitempty"`
	Name          string            `json:"name,omitempty"`
	Kind          string            `json:"kind,omitempty"`
	TimestampMicr int64             `json:"timestamp,omitempty"`
	DurationMicr  int64             `json:"duration,omitempty"`
	LocalEndpoint endpoint          `json:"localEndpoint,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

func EncodeSpans(spans []model.Span) ([]byte, error) {
	out := make([]span, 0, len(spans))
	for _, s := range spans {
		if s.StartTime.IsZero() {
			return nil, fmt.Errorf("span %x has zero start time", s.SpanID)
		}
		d := s.Duration / time.Microsecond
		if d < 1 {
			d = 1
		}
		tags := tagsFromAttrs(s.Attributes)
		if s.Status.Code == "ERROR" {
			tags["error"] = s.Status.Message
			if tags["error"] == "" {
				tags["error"] = "true"
			}
		}
		e := endpoint{ServiceName: serviceName(s)}
		z := span{
			TraceID:       hex.EncodeToString(s.TraceID[:]),
			ID:            hex.EncodeToString(s.SpanID[:]),
			Name:          s.Name,
			Kind:          s.Kind,
			TimestampMicr: s.StartTime.UnixMicro(),
			DurationMicr:  int64(d),
			LocalEndpoint: e,
			Tags:          tags,
		}
		if s.HasParent {
			z.ParentID = hex.EncodeToString(s.ParentSpanID[:])
		}
		out = append(out, z)
	}
	return json.Marshal(out)
}

func tagsFromAttrs(attrs model.Attrs) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string, len(attrs))
	for k, v := range attrs {
		switch tv := v.(type) {
		case string:
			out[k] = tv
		case bool:
			out[k] = strconv.FormatBool(tv)
		case int:
			out[k] = strconv.Itoa(tv)
		case int8, int16, int32, int64:
			out[k] = fmt.Sprintf("%d", tv)
		case uint, uint8, uint16, uint32, uint64:
			out[k] = fmt.Sprintf("%d", tv)
		case float32:
			out[k] = strconv.FormatFloat(float64(tv), 'f', -1, 32)
		case float64:
			out[k] = strconv.FormatFloat(tv, 'f', -1, 64)
		default:
			out[k] = fmt.Sprint(tv)
		}
	}
	return out
}

func serviceName(s model.Span) string {
	if n, ok := s.Resource.Attributes["service.name"].(string); ok && n != "" {
		return n
	}
	if n, ok := s.Attributes["service.name"].(string); ok {
		return n
	}
	return "unknown"
}
