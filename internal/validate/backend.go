package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type tempoClient struct {
	endpoint   string
	httpClient *http.Client
}

func (c tempoClient) Trace(ctx context.Context, traceID string) (traceObservation, error) {
	data, err := getJSON(ctx, c.httpClient, c.endpoint+"/api/traces/"+traceID)
	if err != nil {
		return traceObservation{TraceID: traceID}, err
	}
	obs := observeTempoTrace(traceID, data)
	obs.Found = true
	return obs, nil
}

type jaegerClient struct {
	endpoint   string
	httpClient *http.Client
}

func (c jaegerClient) Trace(ctx context.Context, traceID string) (traceObservation, error) {
	data, err := getJSON(ctx, c.httpClient, c.endpoint+"/api/traces/"+traceID)
	if err != nil {
		return traceObservation{TraceID: traceID}, err
	}
	obs := observeJaegerTrace(traceID, data)
	obs.Found = true
	return obs, nil
}

func getJSON(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("trace not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("backend returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func observeTempoTrace(traceID string, data []byte) traceObservation {
	obs := newObservation(traceID)
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return obs
	}
	walkTempo(payload, &obs)
	return obs
}

func walkTempo(v any, obs *traceObservation) {
	switch x := v.(type) {
	case map[string]any:
		key, _ := x["key"].(string)
		if value, ok := x["value"].(map[string]any); ok && key != "" {
			observeAttribute(obs, key, tempoValue(value))
		}
		if status, ok := x["status"].(map[string]any); ok {
			if code, _ := status["code"].(string); strings.EqualFold(code, "STATUS_CODE_ERROR") {
				obs.ErrorSpans++
			}
		}
		if start, ok := x["startTimeUnixNano"].(string); ok {
			if end, ok := x["endTimeUnixNano"].(string); ok && durationOver100ms(start, end) {
				obs.HighLatencySpan++
			}
		}
		for _, child := range x {
			walkTempo(child, obs)
		}
	case []any:
		for _, child := range x {
			walkTempo(child, obs)
		}
	}
}

func tempoValue(value map[string]any) string {
	for _, key := range []string{"stringValue", "intValue", "doubleValue", "boolValue"} {
		if raw, ok := value[key]; ok {
			return fmt.Sprint(raw)
		}
	}
	return ""
}

func observeJaegerTrace(traceID string, data []byte) traceObservation {
	obs := newObservation(traceID)
	var payload struct {
		Data []struct {
			Processes map[string]struct {
				ServiceName string `json:"serviceName"`
				Tags        []tag  `json:"tags"`
			} `json:"processes"`
			Spans []struct {
				ProcessID string `json:"processID"`
				Duration  int64  `json:"duration"`
				Tags      []tag  `json:"tags"`
			} `json:"spans"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return obs
	}
	for _, trace := range payload.Data {
		for _, process := range trace.Processes {
			if process.ServiceName != "" {
				obs.Services[process.ServiceName] = struct{}{}
			}
			for _, tag := range process.Tags {
				observeAttribute(&obs, tag.Key, fmt.Sprint(tag.Value))
			}
		}
		for _, span := range trace.Spans {
			if process, ok := trace.Processes[span.ProcessID]; ok && process.ServiceName != "" {
				obs.Services[process.ServiceName] = struct{}{}
			}
			if span.Duration > 100_000 {
				obs.HighLatencySpan++
			}
			for _, tag := range span.Tags {
				observeAttribute(&obs, tag.Key, fmt.Sprint(tag.Value))
				if tag.Key == "error" && fmt.Sprint(tag.Value) == "true" {
					obs.ErrorSpans++
				}
			}
		}
	}
	return obs
}

type tag struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func newObservation(traceID string) traceObservation {
	return traceObservation{
		TraceID:  traceID,
		RunIDs:   map[string]struct{}{},
		Services: map[string]struct{}{},
		Phases:   map[string]struct{}{},
	}
}

func observeAttribute(obs *traceObservation, key string, value string) {
	switch key {
	case "service.name":
		if value != "" {
			obs.Services[value] = struct{}{}
		}
	case "spanforge.run_id":
		if value != "" {
			obs.RunIDFound = true
			obs.RunIDs[value] = struct{}{}
		}
	case "spanforge.phase":
		if value != "" {
			obs.Phases[value] = struct{}{}
		}
	case "error":
		if value == "true" {
			obs.ErrorSpans++
		}
	}
}

func durationOver100ms(start, end string) bool {
	var startNano, endNano int64
	if _, err := fmt.Sscan(start, &startNano); err != nil {
		return false
	}
	if _, err := fmt.Sscan(end, &endNano); err != nil {
		return false
	}
	return endNano-startNano > int64(100_000_000)
}
