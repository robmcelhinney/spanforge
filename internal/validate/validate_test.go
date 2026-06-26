package validate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunTempoValidatesReportSamples(t *testing.T) {
	reportPath := writeReport(t, report{
		RunID:          "sf_seed_1",
		Services:       []string{"api-gateway"},
		SampleTraceIDs: []string{"abc123"},
		Phases:         []phaseReport{{Name: "warmup"}},
	})
	client := fakeHTTPClient(func(r *http.Request) (int, string) {
		if r.URL.Path != "/api/traces/abc123" {
			return http.StatusNotFound, ""
		}
		return http.StatusOK, `{
		  "batches": [{
		    "resource": {"attributes": [{"key":"service.name","value":{"stringValue":"api-gateway"}}]},
		    "scopeSpans": [{"spans": [{
		      "startTimeUnixNano": "1000000000",
		      "endTimeUnixNano": "1200000000",
		      "attributes": [
		        {"key":"spanforge.run_id","value":{"stringValue":"sf_seed_1"}},
		        {"key":"spanforge.phase","value":{"stringValue":"warmup"}},
		        {"key":"error","value":{"boolValue":true}}
		      ],
		      "status": {"code":"STATUS_CODE_ERROR"}
		    }]}]
		  }]
		}`
	})

	result, err := Run(context.Background(), Options{
		Backend:      "tempo",
		Endpoint:     "http://tempo.test",
		ReportFile:   reportPath,
		Wait:         time.Millisecond,
		PollInterval: time.Millisecond,
		HTTPClient:   client,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != StatusPass {
		t.Fatalf("status=%s checks=%+v", result.Status, result.Checks)
	}
}

func TestRunJaegerValidatesReportSamples(t *testing.T) {
	reportPath := writeReport(t, report{
		RunID:          "sf_seed_1",
		Services:       []string{"api-gateway"},
		SampleTraceIDs: []string{"abc123"},
	})
	client := fakeHTTPClient(func(r *http.Request) (int, string) {
		if r.URL.Path != "/api/traces/abc123" {
			return http.StatusNotFound, ""
		}
		return http.StatusOK, `{
		  "data": [{
		    "processes": {"p1": {"serviceName":"api-gateway", "tags":[]}},
		    "spans": [{
		      "processID":"p1",
		      "duration":200000,
		      "tags":[
		        {"key":"spanforge.run_id","value":"sf_seed_1"},
		        {"key":"error","value":true}
		      ]
		    }]
		  }]
		}`
	})

	result, err := Run(context.Background(), Options{
		Backend:      "jaeger",
		Endpoint:     "http://jaeger.test",
		ReportFile:   reportPath,
		Wait:         time.Millisecond,
		PollInterval: time.Millisecond,
		HTTPClient:   client,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status == StatusFail {
		t.Fatalf("status=%s checks=%+v", result.Status, result.Checks)
	}
}

func TestRunFailsWhenNoSampleTracesFound(t *testing.T) {
	reportPath := writeReport(t, report{SampleTraceIDs: []string{"missing"}})
	client := fakeHTTPClient(func(r *http.Request) (int, string) {
		return http.StatusNotFound, ""
	})

	result, err := Run(context.Background(), Options{
		Backend:      "tempo",
		Endpoint:     "http://tempo.test",
		ReportFile:   reportPath,
		Wait:         time.Millisecond,
		PollInterval: time.Millisecond,
		HTTPClient:   client,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != StatusFail {
		t.Fatalf("status=%s want fail", result.Status)
	}
}

type roundTripFunc func(*http.Request) (int, string)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	status, body := f(r)
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    r,
	}, nil
}

func fakeHTTPClient(fn func(*http.Request) (int, string)) *http.Client {
	return &http.Client{Transport: roundTripFunc(fn)}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	result := Result{Status: StatusPass, Backend: "tempo", Endpoint: "http://tempo"}
	if err := WriteJSON(&buf, result); err != nil {
		t.Fatalf("write json: %v", err)
	}
	if !strings.Contains(buf.String(), `"status": "pass"`) {
		t.Fatalf("json output=%s", buf.String())
	}
}

func writeReport(t *testing.T, rep report) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	return path
}
