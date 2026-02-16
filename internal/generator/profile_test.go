package generator

import (
	"testing"
	"time"
)

func TestGRPCProfileAddsRPCAttributes(t *testing.T) {
	cfg := baseConfig()
	cfg.Profile = "grpc"
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	if len(trace.Spans) == 0 {
		t.Fatal("expected spans")
	}
	root := trace.Spans[0]
	if root.Attributes["rpc.system"] != "grpc" {
		t.Fatalf("rpc.system=%v want grpc", root.Attributes["rpc.system"])
	}
}

func TestQueueProfileCreatesProducerConsumerWithLink(t *testing.T) {
	cfg := baseConfig()
	cfg.Profile = "queue"
	cfg.Depth = 2
	cfg.Fanout = 2
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	producerFound := false
	consumerFound := false
	linkFound := false
	for _, s := range trace.Spans {
		if s.Kind == "PRODUCER" {
			producerFound = true
			if len(s.Links) > 0 {
				linkFound = true
			}
		}
		if s.Kind == "CONSUMER" {
			consumerFound = true
		}
	}
	if !producerFound || !consumerFound {
		t.Fatalf("expected producer and consumer spans; producer=%v consumer=%v", producerFound, consumerFound)
	}
	if !linkFound {
		t.Fatal("expected producer span link to consumer span")
	}
}

func TestBatchProfileAddsBatchAttributes(t *testing.T) {
	cfg := baseConfig()
	cfg.Profile = "batch"
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	if len(trace.Spans) == 0 {
		t.Fatal("expected spans")
	}
	root := trace.Spans[0]
	if _, ok := root.Attributes["batch.job"]; !ok {
		t.Fatal("expected batch.job attribute")
	}
}

func TestWebProfileHasHTTPStatusCode(t *testing.T) {
	cfg := baseConfig()
	cfg.Profile = "web"
	cfg.Errors = 0
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	if len(trace.Spans) == 0 {
		t.Fatal("expected spans")
	}
	root := trace.Spans[0]
	if code, ok := root.Attributes["http.status_code"].(int); !ok || code != 200 {
		t.Fatalf("http.status_code=%v want 200", root.Attributes["http.status_code"])
	}
}

func TestHighCardinalityAddsIDs(t *testing.T) {
	cfg := baseConfig()
	cfg.HighCardinality = true
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	if len(trace.Spans) == 0 {
		t.Fatal("expected spans")
	}
	root := trace.Spans[0]
	if _, ok := root.Attributes["span.id"]; !ok {
		t.Fatal("expected span.id")
	}
	if _, ok := root.Attributes["trace.id"]; !ok {
		t.Fatal("expected trace.id")
	}
}
