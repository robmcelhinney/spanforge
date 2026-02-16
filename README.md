# spanforge

spanforge is a fake distributed tracing generator for common trace formats such as OTLP (HTTP/gRPC), Zipkin v2 JSON, JSONL, and pretty tree output.

It is useful for testing telemetry pipelines and backends like OpenTelemetry Collector, Tempo, Jaeger, and Zipkin.

> Thanks to [flog](https://github.com/mingrammer/flog) for inspiration.

## Installation

### Using go install

```bash
go install github.com/robmcelhinney/spanforge/cmd/spanforge@latest
```

### Using .tar.gz archive

Download an archive from GitHub Releases (when a tagged release is available), then copy `spanforge` to your system path.

### Using docker

```bash
docker run -it --rm ghcr.io/robmcelhinney/spanforge:latest --help
```

### Build from source

```bash
make build
./bin/spanforge --version
```

## Usage

There are useful options. (`spanforge --help`)

```console
Options:
      --config string               Path to YAML config file
      --rate float                  Generation rate amount (default 200)
      --rate-unit string            Rate unit: spans or traces (default "spans")
      --rate-interval duration      Time interval for rate amount (default 1s)
      --duration duration           Run duration (default 30s)
      --count int                   Total span/trace count (overrides duration if > 0)
      --seed int                    Random seed (default 1)
      --workers int                 Concurrent generator workers (default 1)
      --profile string              Generation profile: web|grpc|queue|batch (default "web")
      --routes int                  Number of named routes/methods/topics/jobs used by profile (default 8)
      --services int                Number of services (default 8)
      --depth int                   Max trace depth (default 4)
      --fanout float                Average span fanout (default 2)
      --service-prefix string       Service name prefix (default "svc-")
      --p50 duration                p50 span latency (default 30ms)
      --p95 duration                p95 span latency (default 120ms)
      --p99 duration                p99 span latency (default 350ms)
      --errors string               Error rate percentage (default "0.5%")
      --retries string              Retry rate percentage (default "1%")
      --db-heavy string             DB-intensive operation ratio (default "20%")
      --cache-hit-rate string       Cache hit ratio (default "85%")
      --variety string              Variety level: low|medium|high (default "medium")
      --high-cardinality            Enable high-cardinality attributes
      --format string               Output format: otlp-http|otlp-grpc|zipkin-json|jsonl|pretty (default "jsonl")
      --output string               Output sink: stdout|file|otlp|zipkin|noop (default "stdout")
      --file string                 Output file path
      --otlp-endpoint string        OTLP endpoint
      --zipkin-endpoint string      Zipkin endpoint
      --headers strings             Additional headers (repeat k=v)
      --compress string             Compression for OTLP HTTP (gzip)
      --batch-size int              Spans per batch (default 512)
      --flush-interval duration     Sink flush interval (default 200ms)
      --sink-retries int            Retry attempts for sink requests (default 2)
      --sink-retry-backoff duration Backoff between sink retries (default 300ms)
      --sink-timeout duration       Per-request sink timeout (default 10s)
      --sink-max-in-flight int      Maximum concurrent in-flight sink requests (default 2)
      --report-file string          Write run summary as JSON to this path
      --http-listen string          Admin HTTP listen address for /healthz and /stats (default "127.0.0.1:8080")
      --version                     Print version and exit
```

```console
# Generate traces to stdout (JSONL)
$ spanforge

# Send OTLP HTTP traces to collector for 2 minutes
$ spanforge --format otlp-http --output otlp --otlp-endpoint http://localhost:4318 --rate 100 --rate-unit traces --duration 2m

# Send Zipkin JSON spans to Zipkin API
$ spanforge --format zipkin-json --output zipkin --zipkin-endpoint http://localhost:9411 --duration 30s

# Run benchmark mode with sink disabled and write JSON report
$ spanforge --output noop --format otlp-http --duration 30s --report-file ./out/report.json

# Load from YAML config, override one value via CLI
$ SPANFORGE_OUTPUT=otlp spanforge --config examples/config/spanforge.yaml --rate 300
```

## Supported Formats

- OTLP HTTP (protobuf)
- OTLP gRPC
- Zipkin v2 JSON
- JSONL
- Pretty tree

## Supported Outputs

- Stdout
- File
- OTLP endpoint
- Zipkin endpoint
- Noop (benchmark mode)

## Environment Variables

`SPANFORGE_*` env vars are supported.

Precedence:

1. CLI flags
2. Environment variables
3. YAML config (`--config`)
4. Built-in defaults

Examples:

- `SPANFORGE_FORMAT=otlp-http`
- `SPANFORGE_OUTPUT=otlp`
- `SPANFORGE_OTLP_ENDPOINT=http://localhost:4318`
- `SPANFORGE_HEADERS=authorization=Bearer token,x-tenant=demo`

## Docker Compose Examples

- Tempo stack: `examples/docker-compose/tempo/docker-compose.yml`
- Jaeger stack: `examples/docker-compose/jaeger/docker-compose.yml`

## Development

```bash
make build
make test
make lint
make bench-transport
```

## Further Reading

Detailed runbooks, presets, and release/CI notes are in `docs/further-info.md`.

## License

[MIT](LICENSE)
