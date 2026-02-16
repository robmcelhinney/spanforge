package app

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
)

type testTraceSvc struct {
	collectortracev1.UnimplementedTraceServiceServer
	spans int
}

func (s *testTraceSvc) Export(_ context.Context, req *collectortracev1.ExportTraceServiceRequest) (*collectortracev1.ExportTraceServiceResponse, error) {
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			s.spans += len(ss.Spans)
		}
	}
	return &collectortracev1.ExportTraceServiceResponse{}, nil
}

func TestRunOTLPGRPC(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("listen unavailable in this environment: %v", err)
	}
	defer lis.Close()

	svc := &testTraceSvc{}
	srv := grpc.NewServer()
	collectortracev1.RegisterTraceServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

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
		Format:           "otlp-grpc",
		Output:           "otlp",
		OTLPEndpoint:     lis.Addr().String(),
		OTLPInsecure:     true,
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
	if svc.spans == 0 {
		t.Fatal("expected spans to be sent")
	}
}
