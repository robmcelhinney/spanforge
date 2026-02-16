package model

import "time"

// TraceID is a 16-byte trace identifier.
type TraceID [16]byte

// SpanID is an 8-byte span identifier.
type SpanID [8]byte

// Attrs stores span/resource attributes.
type Attrs map[string]any

type Resource struct {
	Attributes Attrs
}

type Event struct {
	Name       string
	Time       time.Time
	Attributes Attrs
}

type Link struct {
	TraceID    TraceID
	SpanID     SpanID
	Attributes Attrs
}

type SpanStatus struct {
	Code    string
	Message string
}

type Span struct {
	TraceID      TraceID
	SpanID       SpanID
	ParentSpanID SpanID
	HasParent    bool
	Name         string
	Kind         string
	StartTime    time.Time
	Duration     time.Duration
	Attributes   Attrs
	Events       []Event
	Links        []Link
	Status       SpanStatus
	Resource     Resource
}

type Trace struct {
	TraceID  TraceID
	Resource Resource
	Spans    []Span
}
