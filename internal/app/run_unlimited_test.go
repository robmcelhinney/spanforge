package app

import (
	"context"
	"testing"
	"time"

	"github.com/robmcelhinney/spanforge/internal/config"
	"github.com/robmcelhinney/spanforge/internal/model"
)

func TestProduceTracesUnlimitedDurationStopsOnContextCancel(t *testing.T) {
	cfg := config.Config{
		RateValue:     50,
		RateUnit:      config.RateUnitTraces,
		RateInterval:  time.Second,
		Duration:      0,
		Count:         0,
		Seed:          1,
		Workers:       1,
		Profile:       "web",
		Routes:        2,
		Services:      2,
		Depth:         2,
		Fanout:        1,
		ServicePrefix: "svc-",
		P50:           10 * time.Millisecond,
		P95:           50 * time.Millisecond,
		P99:           80 * time.Millisecond,
		Errors:        0,
		Retries:       0,
		DBHeavy:       0,
		CacheHitRate:  1,
		Variety:       "low",
		BatchSize:     32,
	}

	ctx, cancel := context.WithCancel(context.Background())
	traceCh := make(chan model.Trace, 128)
	done := make(chan error, 1)
	go func() {
		done <- produceTraces(ctx, cfg, traceCh)
		close(traceCh)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("produceTraces returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("produceTraces did not stop after context cancellation")
	}
}
