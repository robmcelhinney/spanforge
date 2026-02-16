package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzEndpoint(t *testing.T) {
	stats := newEmitterStats()
	h := adminHandler(stats)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status=%q want=ok", body["status"])
	}
}

func TestStatsEndpoint(t *testing.T) {
	stats := newEmitterStats()
	stats.add(3, 18)
	h := adminHandler(stats)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}

	var body statsSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.EmittedTraces != 3 {
		t.Fatalf("emitted_traces=%d want=3", body.EmittedTraces)
	}
	if body.EmittedSpans != 18 {
		t.Fatalf("emitted_spans=%d want=18", body.EmittedSpans)
	}
	if body.Status != "ok" {
		t.Fatalf("status=%q want=ok", body.Status)
	}
}
