# spanforge Step-by-Step Action Plan Checklist

Status: implementation checklist. Steps 1-13 track the completed MVP and hardening work from `spanforge_plan.md`; Steps 14 onward track the current roadmap in `ROADMAP.md`.

## Step 1 — Initialize the repo

- [x] Step 1 complete

### Deliverables

- [x] Go module, basic CLI, build works.
- [x] `spanforge --help`, `spanforge --version`.

### Actions

- [x] `go mod init github.com/<you>/spanforge`
- [x] Add CLI framework (cobra or urfave/cli).
- [x] Add `--version` and basic command scaffold.
- [x] Add Makefile.
- [x] Add `make build` target.
- [x] Add `make test` target.
- [x] Add `make lint` target (optional).
- [ ] Add `.golangci.yml` + `golangci-lint` later (optional).

## Step 2 — Define internal model + config

- [x] Step 2 complete

### Deliverables

- [x] `internal/model` with `Trace`, `Span`, `Event`, `Link`, `Resource`.
- [x] `internal/config` with parsed CLI flags into a single `Config` struct.

### Actions

- [x] Define internal types.
- [x] IDs as `[]byte` or fixed-size arrays.
- [x] `time.Time` start + `time.Duration`.
- [x] Attributes as `map[string]any` or typed values.
- [x] Define `Config`.
- [x] Include rate, duration/count, seed, workers.
- [x] Include profile, services, depth, fanout.
- [x] Include p50/p95/p99, errors, retries.
- [x] Include format/output selection + endpoints.
- [x] Implement parsing + validation.
- [x] Parse `--rate`, `--rate-unit`, and `--rate-interval`.
- [x] Parse `%` values.
- [x] Ensure endpoints required for sinks.

## Step 3 — Build the generator core (single profile first)

- [x] Step 3 complete

### Deliverables

- [x] `internal/generator` generates realistic-ish traces with parent/child relationships.
- [x] Deterministic output with a fixed `--seed`.

### Actions

- [x] Implement RNG wrapper seeded from `--seed`.
- [x] Implement topology generator.
- [x] Create N services with names based on `--service-prefix`.
- [x] Choose a fixed “frontdoor” service.
- [x] Implement trace generator.
- [x] Create root SERVER span for request.
- [x] Generate child CLIENT spans for downstream calls.
- [x] Assign `service.name` per span.
- [x] Implement latency distribution.
- [x] Derive lognormal params from p50/p95.
- [x] Sample durations.
- [x] Implement error/retry.
- [x] Inject errors at configured rate.
- [x] If retry triggers, add extra spans + events.

## Step 4 — Add JSONL + Pretty outputs (debug first)

- [x] Step 4 complete

### Deliverables

- [x] `--format jsonl --output stdout|file`
- [x] `--format pretty --output stdout`

### Actions

- [x] Implement JSONL encoder.
- [x] Emit one span per line.
- [x] Include trace_id/span_id/parent_id, timestamps, attrs.
- [x] Implement file sink with rotation optional later.
- [x] Implement pretty printer.
- [x] Group spans by trace_id.
- [x] Print tree with indentation and timing.

## Step 5 — Add rate control + batching

- [x] Step 5 complete

### Deliverables

- [x] `--rate`, `--duration`, `--count`, `--workers` all work.
- [x] Steady generation without huge memory spikes.

### Actions

- [x] Implement trace production loop.
- [x] Add ticker-based scheduling or token bucket (preferred).
- [x] Add a bounded channel between generator and sinks.
- [x] Add batching logic in sinks.
- [x] Accumulate spans up to `--batch-size` or `--flush-interval`.

## Step 6 — Implement OTLP HTTP/protobuf sink (MVP)

- [x] Step 6 complete

### Deliverables

- [x] `--format otlp-http --output otlp --otlp-endpoint http://...:4318`
- [x] Works against OpenTelemetry Collector.

### Actions

- [x] Add OTLP protobuf dependencies.
- [x] `go.opentelemetry.io/proto/otlp/collector/trace/v1`
- [x] `go.opentelemetry.io/proto/otlp/trace/v1`
- [x] `go.opentelemetry.io/proto/otlp/common/v1`
- [x] `go.opentelemetry.io/proto/otlp/resource/v1`
- [x] Write OTLP encoder.
- [x] Map internal model to OTLP `ResourceSpans` -> `ScopeSpans` -> `Span`.
- [x] Implement HTTP client.
- [x] POST to `{endpoint}/v1/traces`.
- [x] Set content-type `application/x-protobuf`.
- [x] Enable gzip if configured.
- [x] Send headers from `--headers`.
- [x] Add integration test.
- [x] Run tiny local OTLP receiver (or collector in CI).
- [x] Assert expected number of spans sent.

## Step 7 — Provide docker + compose quickstart

- [x] Step 7 complete

### Deliverables

- [x] `Dockerfile` builds minimal image.
- [x] `examples/docker-compose/tempo` with collector + tempo.
- [x] README quickstart.

### Actions

- [x] Add multi-stage build.
- [x] Build static binary.
- [x] Copy into distroless image.
- [x] Add compose stack.
- [x] Add otel-collector (4317/4318).
- [x] Add tempo + grafana (optional).
- [x] Add spanforge container pointing to collector.

### Step 7 execution checklist (file-by-file)

- [x] Create `Dockerfile` at repo root.
- [x] Add builder stage with `golang:1.23-alpine` (or current project Go version).
- [x] Use `CGO_ENABLED=0` and `-ldflags="-s -w"` for static binary.
- [x] Copy binary into `gcr.io/distroless/static-debian12:nonroot`.
- [x] Set entrypoint to `/spanforge`.
- [x] Create `examples/docker-compose/tempo/docker-compose.yml`.
- [x] Add `otel-collector` service with mounted config.
- [x] Add `tempo` service and expose `3200`.
- [x] Add optional `grafana` service and expose `3000`.
- [x] Add `spanforge` service using local Dockerfile and OTLP HTTP endpoint `http://otel-collector:4318`.
- [x] Create `examples/docker-compose/tempo/otel-collector-config.yaml`.
- [x] Configure OTLP receiver (`grpc` 4317, `http` 4318), batch processor, and tempo exporter.
- [x] Create README section `Docker Quickstart (Tempo)` with exact commands.
- [x] Verify with `docker compose -f examples/docker-compose/tempo/docker-compose.yml up --build`.
- [x] Verify traces in Grafana Tempo search UI.

### Step 7 acceptance checks

- [x] `docker build -t spanforge:dev .` succeeds.
- [x] `docker run --rm spanforge:dev --version` succeeds.
- [x] `docker compose ... up` starts all services healthy.
- [x] Running spanforge container emits spans with no sink errors.
- [x] Traces visible in Tempo/Grafana.

## Step 8 — Add OTLP gRPC sink

- [ ] Step 8 complete

### Deliverables

- [x] `--format otlp-grpc` works.
- [ ] Comparable performance to HTTP version.

### Actions

- [x] Use OTLP trace service gRPC client.
- [x] Support TLS/insecure.
- [x] Reuse same batching + encoder output with gRPC transport.

## Step 9 — Add Zipkin v2 JSON format + sink

- [x] Step 9 complete

### Deliverables

- [x] `--format zipkin-json --output zipkin --zipkin-endpoint http://zipkin:9411`

### Actions

- [x] Implement Zipkin encoder mapping.
- [x] Map traceId/id/parentId (hex).
- [x] Map timestamp/duration in microseconds.
- [x] Map localEndpoint, tags.
- [x] Implement HTTP sink.
- [x] POST `[]span` batches to `/api/v2/spans`.

## Step 10 — Expand profiles + realism

- [x] Step 10 complete

### Deliverables

- [x] `grpc`, `queue`, `batch` profiles.
- [x] Links/events more common.
- [x] Better naming/attributes for each profile.

### Actions

- [x] Add profile modules.
- [x] Each profile returns span templates + attribute sets.
- [x] Add messaging pattern.
- [x] PRODUCER span with link to CONSUMER.
- [x] Add knobs.
- [x] `--routes`, `--db-heavy`, `--cache-hit-rate` (optional).

## Step 11 — Jaeger support (recommended route first)

- [x] Step 11 complete

### Deliverables

- [x] Documented path: `spanforge -> OTLP -> Collector -> Jaeger`.
- [x] Optional direct Jaeger output later.

### Actions

- [x] Provide compose example with jaeger backend.
- [x] Ensure OTLP spans map cleanly into Jaeger UI.
- [x] If direct Jaeger is needed, implement Jaeger thrift encoder and collector sender later.

## Step 12 — Polish + releases

- [x] Step 12 complete

### Deliverables

- [x] GitHub Actions: build/test, releases, docker multi-arch.
- [x] Performance benchmarks.
- [x] Stable CLI docs + config examples.

### Actions

- [x] Add `goreleaser` for binaries + docker.
- [x] Add `--config` YAML support.
- [x] Add benchmark mode (`--output noop`) and report spans/sec.

## Step 13 — Post-MVP hardening & operability

- [x] Step 13 complete

### Deliverables

- [x] OTLP HTTP vs OTLP gRPC perf comparison harness with repeatable output.
- [x] CI docker-compose smoke tests for Tempo and Jaeger examples.
- [x] Env var config support with precedence `CLI > env > YAML`.
- [x] Sink resiliency knobs (retry/backoff/timeouts/max in-flight).
- [x] Structured benchmark report output (`--report-file` JSON).
- [x] Release docs for tag flow + required GitHub secrets/permissions.

### Actions

- [x] Add transport perf comparison harness for OTLP HTTP vs gRPC.
- [x] Add CI docker-compose smoke tests for Tempo and Jaeger examples.
- [x] Add env var config support and precedence rules.
- [x] Add sink retry/backoff/timeout/in-flight controls.
- [x] Add `--report-file` JSON benchmark output.
- [x] Document release publishing requirements and process.

## Step 14 — Credible named profiles

- [x] Step 14 complete

### Deliverables

- [x] `spanforge profiles list`
- [x] `spanforge profiles show <name>`
- [x] `payment-system` profile with recognizable checkout/payment traces.
- [x] `api-gateway` profile with shallow high-volume request traces.
- [x] Deterministic `--run-id`.
- [x] `spanforge.run_id`, `spanforge.profile`, and `spanforge.seed` attributes on spans.
- [x] Expanded `--report-file` manifest with services and sample trace IDs.
- [x] README realistic profile examples and screenshots.

### Actions

- [x] Introduce a profile registry that can describe profiles as well as generate traces.
- [x] Decide whether to evolve `internal/generator/profile.go` or split profiles into `internal/profile`.
- [x] Add profile metadata: name, description, services, routes, and failure modes.
- [x] Add CLI subcommands for profile listing and detail output.
- [x] Add `payment-system` services, routes, attributes, and failure scenarios.
- [x] Add `api-gateway` services, routes, status distribution, auth failures, and rate-limit failures.
- [x] Include run metadata on every generated span.
- [x] Capture sample trace IDs and service names in the run report.
- [x] Add focused tests for deterministic profile output.

## Step 15 — Load phases

- [x] Step 15 complete

### Deliverables

- [x] `--phase-file` YAML support.
- [x] `--load` built-ins: `steady`, `warmup-spike-recovery`, `brownout`, `sawtooth`, `error-storm`.
- [x] `spanforge.phase` attribute on spans.
- [x] Phase totals in `--report-file`.
- [x] Docs recipe for simulating a checkout brownout.

### Actions

- [x] Define phase config schema.
- [x] Validate phase duration, rate, rate unit, latency, error, and retry overrides.
- [x] Reuse the existing production loop for sequential phases.
- [x] Apply phase-specific config without breaking deterministic seeds.
- [x] Add tests for phase ordering, counts, and metadata.

## Step 16 — Tempo/Grafana demo stack

- [x] Step 16 complete

### Deliverables

- [x] `examples/docker-compose/tempo-grafana`.
- [x] Pre-provisioned Grafana datasource.
- [x] Dashboard JSON for recent traces, error traces, OK traces, and duration thresholds.
- [x] README screenshots or GIFs.

### Actions

- [x] Create compose stack with collector, Tempo, Grafana, and spanforge.
- [x] Provision Grafana datasource automatically.
- [x] Add dashboard panels for recent traces, error traces, OK traces, and duration thresholds.
- [x] Document one-command demo flow.
- [x] Capture screenshots after the stack is verified.

## Step 17 — Backend validation

- [ ] Step 17 complete

### Deliverables

- [ ] `spanforge validate tempo`.
- [ ] `spanforge validate jaeger`.
- [ ] `--wait` and `--poll-interval`.
- [ ] `--output text|json`.
- [ ] Validation using `spanforge.run_id` and sample trace IDs from `--report-file`.

### Actions

- [ ] Define validation result schema with pass, warn, and fail states.
- [ ] Add Tempo query client.
- [ ] Add Jaeger query client.
- [ ] Check run ID, expected services, sampled traces, error spans, high-latency spans, and phase labels.
- [ ] Avoid exact trace-count promises by default.
- [ ] Add integration tests where local backend APIs are practical.

## Step 18 — Weird and invalid telemetry lab

- [ ] Step 18 complete

### Deliverables

- [ ] `--weird clock-skew,huge-duration,future-timestamp,high-cardinality-route,huge-attribute,mixed-semconv`.
- [ ] `--invalid duplicate-span-id,negative-duration,bad-encoded-payload,empty-required-fields`.
- [ ] Docs explaining what each mode tests and whether ingestion should usually succeed.
- [ ] Demo dashboard panels for weird traces.

### Actions

- [ ] Add config parsing and validation for `--weird` and `--invalid`.
- [ ] Implement weird modes as valid trace mutations where possible.
- [ ] Keep invalid modes explicit and hard to trigger accidentally.
- [ ] Document backend-specific expected behavior.
- [ ] Add tests that distinguish valid weird telemetry from intentionally invalid payloads.

## Step 19 — Serious OSS release

- [ ] Step 19 complete

### Deliverables

- [ ] Published GitHub releases.
- [ ] GHCR Docker images.
- [ ] Compatibility matrix for OpenTelemetry Collector, Tempo, Jaeger, and Zipkin.
- [ ] Stable profile names.
- [ ] Stable report JSON schema.
- [ ] Stable validation JSON schema.
- [ ] Contribution guide and issue templates.
- [ ] Homebrew tap when release usage justifies it.

### Actions

- [ ] Verify release workflow against a real tag.
- [ ] Publish Docker image tags for latest and versioned releases.
- [ ] Add backend compatibility docs.
- [ ] Mark stable schemas and document migration policy.
- [ ] Add contribution and issue templates.
