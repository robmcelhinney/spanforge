package validate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Options struct {
	Backend      string
	Endpoint     string
	ReportFile   string
	Wait         time.Duration
	PollInterval time.Duration
	HTTPClient   *http.Client
}

type Result struct {
	Status   Status  `json:"status"`
	Backend  string  `json:"backend"`
	Endpoint string  `json:"endpoint"`
	Checks   []Check `json:"checks"`
}

type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

type report struct {
	RunID          string        `json:"run_id"`
	Services       []string      `json:"services"`
	SampleTraceIDs []string      `json:"sample_trace_ids"`
	Phases         []phaseReport `json:"phases"`
}

type phaseReport struct {
	Name string `json:"name"`
}

type traceObservation struct {
	TraceID         string
	Found           bool
	RunIDFound      bool
	RunIDs          map[string]struct{}
	Services        map[string]struct{}
	Phases          map[string]struct{}
	ErrorSpans      int
	HighLatencySpan int
}

type backendClient interface {
	Trace(ctx context.Context, traceID string) (traceObservation, error)
}

func Run(ctx context.Context, opts Options) (Result, error) {
	opts.Backend = strings.ToLower(strings.TrimSpace(opts.Backend))
	if opts.Wait <= 0 {
		opts.Wait = 30 * time.Second
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(opts.ReportFile) == "" {
		return Result{}, errors.New("report-file is required")
	}
	rep, err := loadReport(opts.ReportFile)
	if err != nil {
		return Result{}, err
	}
	client, endpoint, err := newBackendClient(opts)
	if err != nil {
		return Result{}, err
	}

	deadline := time.Now().Add(opts.Wait)
	var observations []traceObservation
	var lastErr error
	for {
		observations, lastErr = fetchSamples(ctx, client, rep.SampleTraceIDs)
		if foundAny(observations) || time.Now().After(deadline) {
			break
		}
		timer := time.NewTimer(opts.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return Result{}, ctx.Err()
		case <-timer.C:
		}
	}

	result := Result{
		Status:   StatusPass,
		Backend:  opts.Backend,
		Endpoint: endpoint,
		Checks:   buildChecks(rep, observations, lastErr),
	}
	for _, check := range result.Checks {
		switch check.Status {
		case StatusFail:
			result.Status = StatusFail
		case StatusWarn:
			if result.Status == StatusPass {
				result.Status = StatusWarn
			}
		}
	}
	return result, nil
}

func WriteText(w io.Writer, result Result) error {
	if _, err := fmt.Fprintf(w, "validation %s: %s (%s)\n", result.Status, result.Backend, result.Endpoint); err != nil {
		return err
	}
	for _, check := range result.Checks {
		if _, err := fmt.Fprintf(w, "- %s: %s: %s\n", check.Status, check.Name, check.Message); err != nil {
			return err
		}
	}
	return nil
}

func WriteJSON(w io.Writer, result Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func loadReport(path string) (report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return report{}, fmt.Errorf("read report file: %w", err)
	}
	var rep report
	if err := json.Unmarshal(data, &rep); err != nil {
		return report{}, fmt.Errorf("parse report file: %w", err)
	}
	if len(rep.SampleTraceIDs) == 0 {
		return report{}, fmt.Errorf("report file has no sample_trace_ids")
	}
	return rep, nil
}

func newBackendClient(opts Options) (backendClient, string, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(opts.Endpoint), "/")
	switch opts.Backend {
	case "tempo":
		if endpoint == "" {
			endpoint = "http://localhost:3200"
		}
		return tempoClient{endpoint: endpoint, httpClient: opts.HTTPClient}, endpoint, nil
	case "jaeger":
		if endpoint == "" {
			endpoint = "http://localhost:16686"
		}
		return jaegerClient{endpoint: endpoint, httpClient: opts.HTTPClient}, endpoint, nil
	default:
		return nil, "", fmt.Errorf("unsupported validation backend %q", opts.Backend)
	}
}

func fetchSamples(ctx context.Context, client backendClient, traceIDs []string) ([]traceObservation, error) {
	observations := make([]traceObservation, 0, len(traceIDs))
	var lastErr error
	for _, traceID := range traceIDs {
		obs, err := client.Trace(ctx, traceID)
		if err != nil {
			lastErr = err
			obs = traceObservation{TraceID: traceID}
		}
		observations = append(observations, obs)
	}
	return observations, lastErr
}

func foundAny(observations []traceObservation) bool {
	for _, obs := range observations {
		if obs.Found {
			return true
		}
	}
	return false
}

func buildChecks(rep report, observations []traceObservation, lastErr error) []Check {
	checks := []Check{}
	found := 0
	for _, obs := range observations {
		if obs.Found {
			found++
		}
	}
	switch {
	case found == len(observations):
		checks = append(checks, Check{"sample_traces", StatusPass, fmt.Sprintf("found all %d sampled traces", found)})
	case found > 0:
		checks = append(checks, Check{"sample_traces", StatusWarn, fmt.Sprintf("found %d of %d sampled traces", found, len(observations))})
	default:
		msg := fmt.Sprintf("found 0 of %d sampled traces", len(observations))
		if lastErr != nil {
			msg += ": " + lastErr.Error()
		}
		checks = append(checks, Check{"sample_traces", StatusFail, msg})
	}

	runIDMatches := countRunIDMatches(observations, rep.RunID)
	if rep.RunID == "" {
		checks = append(checks, Check{"run_id", StatusWarn, "report has no run_id to validate"})
	} else if runIDMatches > 0 {
		checks = append(checks, Check{"run_id", StatusPass, fmt.Sprintf("found spanforge.run_id=%q in sampled traces", rep.RunID)})
	} else {
		checks = append(checks, Check{"run_id", StatusWarn, fmt.Sprintf("sampled traces did not expose spanforge.run_id=%q", rep.RunID)})
	}

	observedServices := collectSet(observations, func(obs traceObservation) map[string]struct{} { return obs.Services })
	missingServices := missing(rep.Services, observedServices)
	if len(rep.Services) == 0 {
		checks = append(checks, Check{"services", StatusWarn, "report has no services to validate"})
	} else if len(missingServices) == 0 {
		checks = append(checks, Check{"services", StatusPass, fmt.Sprintf("found expected services: %s", strings.Join(rep.Services, ", "))})
	} else {
		checks = append(checks, Check{"services", StatusWarn, fmt.Sprintf("missing services in sampled traces: %s", strings.Join(missingServices, ", "))})
	}

	expectedPhases := phaseNames(rep.Phases)
	observedPhases := collectSet(observations, func(obs traceObservation) map[string]struct{} { return obs.Phases })
	missingPhases := missing(expectedPhases, observedPhases)
	if len(expectedPhases) == 0 {
		checks = append(checks, Check{"phase_labels", StatusPass, "report has no phases to validate"})
	} else if len(missingPhases) == 0 {
		checks = append(checks, Check{"phase_labels", StatusPass, fmt.Sprintf("found expected phases: %s", strings.Join(expectedPhases, ", "))})
	} else {
		checks = append(checks, Check{"phase_labels", StatusWarn, fmt.Sprintf("missing phase labels in sampled traces: %s", strings.Join(missingPhases, ", "))})
	}

	errorSpans, highLatencySpans := 0, 0
	for _, obs := range observations {
		errorSpans += obs.ErrorSpans
		highLatencySpans += obs.HighLatencySpan
	}
	if errorSpans > 0 {
		checks = append(checks, Check{"error_spans", StatusPass, fmt.Sprintf("found %d sampled error spans", errorSpans)})
	} else {
		checks = append(checks, Check{"error_spans", StatusWarn, "no error spans found in sampled traces"})
	}
	if highLatencySpans > 0 {
		checks = append(checks, Check{"high_latency_spans", StatusPass, fmt.Sprintf("found %d sampled spans over 100ms", highLatencySpans)})
	} else {
		checks = append(checks, Check{"high_latency_spans", StatusWarn, "no sampled spans over 100ms"})
	}

	return checks
}

func countRunIDMatches(observations []traceObservation, expected string) int {
	count := 0
	for _, obs := range observations {
		if _, ok := obs.RunIDs[expected]; ok || (expected == "" && obs.RunIDFound) {
			count++
		}
	}
	return count
}

func collectSet(observations []traceObservation, fn func(traceObservation) map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for _, obs := range observations {
		for v := range fn(obs) {
			out[v] = struct{}{}
		}
	}
	return out
}

func missing(expected []string, observed map[string]struct{}) []string {
	out := []string{}
	for _, item := range expected {
		if _, ok := observed[item]; !ok {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func phaseNames(phases []phaseReport) []string {
	out := make([]string, 0, len(phases))
	for _, phase := range phases {
		if phase.Name != "" {
			out = append(out, phase.Name)
		}
	}
	sort.Strings(out)
	return out
}
