package otlphttp

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robmcelhinney/spanforge/internal/encode/otlp"
	"github.com/robmcelhinney/spanforge/internal/model"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	endpoint string
	headers  map[string]string
	gzip     bool
	http     *http.Client
}

func New(endpoint string, headers map[string]string, gzipEnabled bool, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		headers:  headers,
		gzip:     gzipEnabled,
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) SendSpans(ctx context.Context, spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	reqMsg, err := otlp.EncodeSpans(spans)
	if err != nil {
		return err
	}
	payload, err := proto.Marshal(reqMsg)
	if err != nil {
		return fmt.Errorf("marshal otlp request: %w", err)
	}
	body := payload

	reqBody := io.Reader(bytes.NewReader(body))
	if c.gzip {
		var gzBuf bytes.Buffer
		zw := gzip.NewWriter(&gzBuf)
		if _, err := zw.Write(body); err != nil {
			return err
		}
		if err := zw.Close(); err != nil {
			return err
		}
		reqBody = bytes.NewReader(gzBuf.Bytes())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/traces", reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	if c.gzip {
		req.Header.Set("Content-Encoding", "gzip")
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("otlp http error: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return nil
}
