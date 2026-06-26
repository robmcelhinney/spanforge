package generator

import (
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/model"
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

func TestPaymentSystemProfileAddsPaymentAttributes(t *testing.T) {
	cfg := baseConfig()
	cfg.Profile = "payment-system"
	cfg.Depth = 2
	cfg.Fanout = 7
	cfg.Routes = 7
	trace := New(cfg).GenerateTrace(time.Now().UTC())
	if len(trace.Spans) == 0 {
		t.Fatal("expected spans")
	}
	root := trace.Spans[0]
	if root.Attributes["service.name"] != "edge-gateway" {
		t.Fatalf("root service=%v want edge-gateway", root.Attributes["service.name"])
	}
	foundProvider := false
	foundLedger := false
	foundFraud := false
	for _, span := range trace.Spans {
		if _, ok := span.Attributes["payment.provider"]; ok {
			foundProvider = true
		}
		if _, ok := span.Attributes["ledger.account_type"]; ok {
			foundLedger = true
		}
		if _, ok := span.Attributes["fraud.score_bucket"]; ok {
			foundFraud = true
		}
	}
	if !foundProvider || !foundLedger || !foundFraud {
		t.Fatalf("missing payment profile attrs provider=%v ledger=%v fraud=%v", foundProvider, foundLedger, foundFraud)
	}
}

func TestAPIGatewayProfileAddsGatewayAttributes(t *testing.T) {
	profile := apiGatewayProfile{}
	now := time.Now().UTC()
	traceID := modelTraceID(1)
	root := profile.buildRoot("gateway", 0, now, modelSpanID(1), traceID, time.Millisecond)
	if root.Attributes["service.name"] != "gateway" {
		t.Fatalf("root service=%v want gateway", root.Attributes["service.name"])
	}
	auth := profile.buildChild(root, "", 0, now, modelSpanID(2), traceID, time.Millisecond, false, true)
	rateLimit := profile.buildChild(root, "", 1, now, modelSpanID(3), traceID, time.Millisecond, false, true)
	upstream := profile.buildChild(root, "", 2, now, modelSpanID(4), traceID, time.Millisecond, false, true)
	if _, ok := auth.Attributes["auth.result"]; !ok {
		t.Fatal("expected auth.result")
	}
	if _, ok := rateLimit.Attributes["rate_limit.decision"]; !ok {
		t.Fatal("expected rate_limit.decision")
	}
	if _, ok := upstream.Attributes["upstream.service"]; !ok {
		t.Fatal("expected upstream.service")
	}
}

func TestProfileRegistryIncludesNamedProfiles(t *testing.T) {
	if _, ok := Profile("payment-system"); !ok {
		t.Fatal("expected payment-system profile metadata")
	}
	if _, ok := Profile("api-gateway"); !ok {
		t.Fatal("expected api-gateway profile metadata")
	}
}

func modelTraceID(seed byte) model.TraceID {
	var id model.TraceID
	id[0] = seed
	return id
}

func modelSpanID(seed byte) model.SpanID {
	var id model.SpanID
	id[0] = seed
	return id
}
