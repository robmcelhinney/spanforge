package app

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestRunOTLPHTTP(t *testing.T) {
	var gotSpans int
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("listen unavailable in this environment: %v", err)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" {
			t.Fatalf("path=%s want /v1/traces", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
			t.Fatalf("content-type=%q", ct)
		}
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req collectortracev1.ExportTraceServiceRequest
		if err := proto.Unmarshal(payload, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		for _, rs := range req.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				gotSpans += len(ss.Spans)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	srv.Listener = lis
	srv.Start()
	defer srv.Close()

	cfg := config.Config{
		RateValue:        100,
		RateUnit:         config.RateUnitTraces,
		RateInterval:     time.Second,
		Duration:         1 * time.Second,
		Count:            2,
		Seed:             1,
		Workers:          1,
		Profile:          "web",
		Routes:           2,
		Services:         3,
		Depth:            2,
		Fanout:           1,
		ServicePrefix:    "svc-",
		P50:              10 * time.Millisecond,
		P95:              50 * time.Millisecond,
		P99:              80 * time.Millisecond,
		Errors:           0,
		Retries:          0,
		CacheHitRate:     1,
		Format:           "otlp-http",
		Output:           "otlp",
		OTLPEndpoint:     srv.URL,
		BatchSize:        32,
		FlushInterval:    100 * time.Millisecond,
		SinkRetries:      0,
		SinkRetryBackoff: 100 * time.Millisecond,
		SinkTimeout:      time.Second,
		SinkMaxInFlight:  1,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotSpans == 0 {
		t.Fatal("expected spans to be sent")
	}
}
