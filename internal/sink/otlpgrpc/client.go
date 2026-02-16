package otlpgrpc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robmcelhinney/spanforge/internal/encode/otlp"
	"github.com/robmcelhinney/spanforge/internal/model"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	endpoint string
	headers  map[string]string
	insecure bool
	timeout  time.Duration

	mu     sync.Mutex
	conn   *grpc.ClientConn
	client collectortracev1.TraceServiceClient
}

func New(endpoint string, headers map[string]string, insecureConn bool, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		endpoint: strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://"),
		headers:  headers,
		insecure: insecureConn,
		timeout:  timeout,
	}
}

func (c *Client) SendSpans(ctx context.Context, spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}
	cli, err := c.ensureClient(ctx)
	if err != nil {
		return err
	}

	req, err := otlp.EncodeSpans(spans)
	if err != nil {
		return err
	}

	callCtx := ctx
	if len(c.headers) > 0 {
		md := metadata.New(c.headers)
		callCtx = metadata.NewOutgoingContext(callCtx, md)
	}
	callCtx, cancel := context.WithTimeout(callCtx, c.timeout)
	defer cancel()

	if _, err := cli.Export(callCtx, req); err != nil {
		return fmt.Errorf("otlp grpc export: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) ensureClient(ctx context.Context) (collectortracev1.TraceServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client, nil
	}
	if c.endpoint == "" {
		return nil, fmt.Errorf("empty OTLP gRPC endpoint")
	}

	var creds credentials.TransportCredentials
	if c.insecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewClientTLSFromCert(nil, "")
	}
	dialCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, c.endpoint, grpc.WithTransportCredentials(creds), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	c.conn = conn
	c.client = collectortracev1.NewTraceServiceClient(conn)
	return c.client, nil
}
