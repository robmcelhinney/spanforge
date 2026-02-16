package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"time"
)

type emitterStats struct {
	startedAt time.Time
	traces    uint64
	spans     uint64
}

type statsSnapshot struct {
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
	UptimeSeconds float64   `json:"uptime_seconds"`
	EmittedTraces uint64    `json:"emitted_traces"`
	EmittedSpans  uint64    `json:"emitted_spans"`
}

func newEmitterStats() *emitterStats {
	return &emitterStats{startedAt: time.Now().UTC()}
}

func (s *emitterStats) add(traces, spans int) {
	if traces > 0 {
		atomic.AddUint64(&s.traces, uint64(traces))
	}
	if spans > 0 {
		atomic.AddUint64(&s.spans, uint64(spans))
	}
}

func (s *emitterStats) snapshot() statsSnapshot {
	now := time.Now().UTC()
	return statsSnapshot{
		Status:        "ok",
		StartedAt:     s.startedAt,
		UptimeSeconds: now.Sub(s.startedAt).Seconds(),
		EmittedTraces: atomic.LoadUint64(&s.traces),
		EmittedSpans:  atomic.LoadUint64(&s.spans),
	}
}

func adminHandler(stats *emitterStats) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats.snapshot())
	})
	return mux
}

func runAdminServer(ctx context.Context, listenAddr string, stats *emitterStats) error {
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: adminHandler(stats),
	}

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	err := srv.ListenAndServe()
	<-shutdownDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
