package app

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type perfTraceSvc struct {
	collectortracev1.UnimplementedTraceServiceServer
	spans int
}

func (s *perfTraceSvc) Export(_ context.Context, req *collectortracev1.ExportTraceServiceRequest) (*collectortracev1.ExportTraceServiceResponse, error) {
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			s.spans += len(ss.Spans)
		}
	}
	return &collectortracev1.ExportTraceServiceResponse{}, nil
}

func TestTransportPerfComparison(t *testing.T) {
	if os.Getenv("SPANFORGE_TRANSPORT_PERF") != "1" {
		t.Skip("set SPANFORGE_TRANSPORT_PERF=1 to run transport perf comparison")
	}

	httpSpans, httpDur := runPerfHTTP(t)
	grpcSpans, grpcDur := runPerfGRPC(t)

	httpSPS := float64(httpSpans) / httpDur.Seconds()
	grpcSPS := float64(grpcSpans) / grpcDur.Seconds()
	ratio := grpcSPS / httpSPS

	t.Logf("transport perf: http spans=%d duration=%s spans/sec=%.2f", httpSpans, httpDur, httpSPS)
	t.Logf("transport perf: grpc spans=%d duration=%s spans/sec=%.2f", grpcSpans, grpcDur, grpcSPS)
	t.Logf("transport perf ratio grpc/http=%.3f", ratio)

	if ratio < 0.40 {
		t.Fatalf("grpc throughput too low vs http: ratio=%.3f", ratio)
	}
}

func runPerfHTTP(t *testing.T) (int, time.Duration) {
	t.Helper()
	var gotSpans int
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("listen unavailable: %v", err)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	cfg := perfConfig()
	cfg.Format = "otlp-http"
	cfg.Output = "otlp"
	cfg.OTLPEndpoint = srv.URL

	start := time.Now()
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run http: %v", err)
	}
	return gotSpans, time.Since(start)
}

func runPerfGRPC(t *testing.T) (int, time.Duration) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("listen unavailable: %v", err)
	}
	defer lis.Close()

	svc := &perfTraceSvc{}
	srv := grpc.NewServer()
	collectortracev1.RegisterTraceServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	cfg := perfConfig()
	cfg.Format = "otlp-grpc"
	cfg.Output = "otlp"
	cfg.OTLPEndpoint = lis.Addr().String()
	cfg.OTLPInsecure = true

	start := time.Now()
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run grpc: %v", err)
	}
	return svc.spans, time.Since(start)
}

func perfConfig() config.Config {
	return config.Config{
		RateValue:        100000,
		RateUnit:         config.RateUnitTraces,
		RateInterval:     time.Second,
		Duration:         2 * time.Second,
		Count:            200,
		Seed:             1,
		Workers:          2,
		Profile:          "web",
		Routes:           8,
		Services:         8,
		Depth:            3,
		Fanout:           1.5,
		ServicePrefix:    "svc-",
		P50:              10 * time.Millisecond,
		P95:              50 * time.Millisecond,
		P99:              120 * time.Millisecond,
		Errors:           0,
		Retries:          0,
		DBHeavy:          0.2,
		CacheHitRate:     0.8,
		Variety:          "medium",
		HighCardinality:  false,
		BatchSize:        256,
		FlushInterval:    100 * time.Millisecond,
		SinkRetries:      0,
		SinkRetryBackoff: 100 * time.Millisecond,
		SinkTimeout:      2 * time.Second,
		SinkMaxInFlight:  2,
		HTTPListen:       "",
	}
}
