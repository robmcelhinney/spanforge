package otlp

import (
	"fmt"
	"sort"
	"time"

	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/robmcelhinney/spanforge/internal/model"
)

func EncodeSpans(spans []model.Span) (*collectortracev1.ExportTraceServiceRequest, error) {
	byService := map[string][]model.Span{}
	for _, s := range spans {
		service, _ := s.Attributes["service.name"].(string)
		if service == "" {
			service = "unknown-service"
		}
		byService[service] = append(byService[service], s)
	}

	services := make([]string, 0, len(byService))
	for k := range byService {
		services = append(services, k)
	}
	sort.Strings(services)

	resourceSpans := make([]*tracev1.ResourceSpans, 0, len(services))
	for _, service := range services {
		spansForService := byService[service]
		otelSpans := make([]*tracev1.Span, 0, len(spansForService))
		for _, s := range spansForService {
			span, err := toOTLPSpan(s)
			if err != nil {
				return nil, err
			}
			otelSpans = append(otelSpans, span)
		}

		resourceSpans = append(resourceSpans, &tracev1.ResourceSpans{
			Resource: &resourcev1.Resource{Attributes: []*commonv1.KeyValue{{
				Key:   "service.name",
				Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: service}},
			}}},
			ScopeSpans: []*tracev1.ScopeSpans{{Spans: otelSpans}},
		})
	}

	return &collectortracev1.ExportTraceServiceRequest{ResourceSpans: resourceSpans}, nil
}

func toOTLPSpan(s model.Span) (*tracev1.Span, error) {
	start := s.StartTime.UTC()
	end := s.StartTime.Add(s.Duration).UTC()
	if s.Duration < 0 {
		return nil, fmt.Errorf("negative duration for span %q", s.Name)
	}

	attrs := toAttrs(s.Attributes)
	events := make([]*tracev1.Span_Event, 0, len(s.Events))
	for _, e := range s.Events {
		events = append(events, &tracev1.Span_Event{
			Name:                   e.Name,
			TimeUnixNano:           uint64(e.Time.UTC().UnixNano()),
			Attributes:             toAttrs(e.Attributes),
			DroppedAttributesCount: 0,
		})
	}

	span := &tracev1.Span{
		TraceId:           append([]byte(nil), s.TraceID[:]...),
		SpanId:            append([]byte(nil), s.SpanID[:]...),
		Name:              s.Name,
		Kind:              toSpanKind(s.Kind),
		StartTimeUnixNano: uint64(start.UnixNano()),
		EndTimeUnixNano:   uint64(end.UnixNano()),
		Attributes:        attrs,
		Events:            events,
		Status:            toStatus(s.Status.Code, s.Status.Message),
	}
	if s.HasParent {
		span.ParentSpanId = append([]byte(nil), s.ParentSpanID[:]...)
	}
	return span, nil
}

func toSpanKind(kind string) tracev1.Span_SpanKind {
	switch kind {
	case "INTERNAL":
		return tracev1.Span_SPAN_KIND_INTERNAL
	case "SERVER":
		return tracev1.Span_SPAN_KIND_SERVER
	case "CLIENT":
		return tracev1.Span_SPAN_KIND_CLIENT
	case "PRODUCER":
		return tracev1.Span_SPAN_KIND_PRODUCER
	case "CONSUMER":
		return tracev1.Span_SPAN_KIND_CONSUMER
	default:
		return tracev1.Span_SPAN_KIND_UNSPECIFIED
	}
}

func toStatus(code, message string) *tracev1.Status {
	out := &tracev1.Status{Message: message}
	switch code {
	case "OK":
		out.Code = tracev1.Status_STATUS_CODE_OK
	case "ERROR":
		out.Code = tracev1.Status_STATUS_CODE_ERROR
	default:
		out.Code = tracev1.Status_STATUS_CODE_UNSET
	}
	return out
}

func toAttrs(attrs model.Attrs) []*commonv1.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*commonv1.KeyValue, 0, len(keys))
	for _, k := range keys {
		out = append(out, &commonv1.KeyValue{Key: k, Value: toAny(attrs[k])})
	}
	return out
}

func toAny(v any) *commonv1.AnyValue {
	switch t := v.(type) {
	case string:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: t}}
	case bool:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: t}}
	case int:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: int64(t)}}
	case int64:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: t}}
	case float64:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: t}}
	case float32:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: float64(t)}}
	case time.Duration:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: float64(t) / float64(time.Millisecond)}}
	default:
		return &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: fmt.Sprint(v)}}
	}
}
