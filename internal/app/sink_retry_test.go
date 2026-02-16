package app

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
)

func TestRunRetriesSinkRequest(t *testing.T) {
	var calls int32
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("listen unavailable in this environment: %v", err)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		_, _ = io.Copy(io.Discard, r.Body)
		if n == 1 {
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
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
		Duration:         time.Second,
		Count:            1,
		Seed:             1,
		Workers:          1,
		Profile:          "web",
		Routes:           2,
		Services:         2,
		Depth:            2,
		Fanout:           1,
		ServicePrefix:    "svc-",
		P50:              10 * time.Millisecond,
		P95:              50 * time.Millisecond,
		P99:              80 * time.Millisecond,
		Errors:           0,
		Retries:          0,
		DBHeavy:          0,
		CacheHitRate:     1,
		Variety:          "low",
		HighCardinality:  false,
		Format:           "otlp-http",
		Output:           "otlp",
		OTLPEndpoint:     srv.URL,
		BatchSize:        32,
		FlushInterval:    100 * time.Millisecond,
		SinkRetries:      1,
		SinkRetryBackoff: 10 * time.Millisecond,
		SinkTimeout:      time.Second,
		SinkMaxInFlight:  1,
		HTTPListen:       "",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := Run(cfg, bytes.NewBuffer(nil)); err != nil {
		t.Fatalf("run: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected retry call count >=2, got %d", calls)
	}
}
