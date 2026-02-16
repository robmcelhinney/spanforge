package zipkin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/robmcelhinney/spanforge/internal/encode/zipkin"
	"github.com/robmcelhinney/spanforge/internal/model"
)

type Client struct {
	endpoint string
	headers  map[string]string
	http     *http.Client
}

func New(endpoint string, headers map[string]string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		endpoint: endpoint,
		headers:  headers,
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) SendSpans(ctx context.Context, spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	payload, err := zipkin.EncodeSpans(spans)
	if err != nil {
		return fmt.Errorf("encode zipkin spans: %w", err)
	}
	endpoint, err := zipkinSpansURL(c.endpoint)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("zipkin export request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("zipkin http error: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return nil
}

func zipkinSpansURL(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid zipkin endpoint %q: %w", base, err)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Host == "" {
		u.Host = u.Path
		u.Path = ""
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/api/v2/spans"
	}
	return u.String(), nil
}
