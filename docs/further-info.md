# spanforge Further Info

This page keeps detailed operational guidance, presets, and release notes that are intentionally not in the top-level `README.md`.

## Config File (YAML)

Load settings from a YAML file:

```bash
./bin/spanforge --config examples/config/spanforge.yaml
```

CLI flags override YAML values when both are set.

Environment variables are also supported using `SPANFORGE_*` names.
Precedence is: `CLI flags > env vars > YAML config > built-in defaults`.

Set `duration: 0s` (or `--duration 0s`) to run indefinitely with no duration limit.

Examples:

- `SPANFORGE_FORMAT=otlp-http`
- `SPANFORGE_OUTPUT=otlp`
- `SPANFORGE_OTLP_ENDPOINT=http://localhost:4318`
- `SPANFORGE_HEADERS=authorization=Bearer token,x-tenant=demo`
- `SPANFORGE_DEBUG=true`

## Zipkin Output

Send traces directly to Zipkin:

```bash
./bin/spanforge \
  --format zipkin-json \
  --output zipkin \
  --zipkin-endpoint http://localhost:9411 \
  --rate 20 \
  --rate-unit traces \
  --rate-interval 1s \
  --duration 30s
```

## Jaeger via OTLP Collector

Recommended path: `spanforge -> otel-collector -> jaeger`.

1. Start stack:

```bash
docker compose -f examples/docker-compose/jaeger/docker-compose.yml up --build
```

2. Open Jaeger UI:

```text
http://localhost:16686
```

3. In Jaeger Search, choose `service=api-gateway` and run query.

## Preset Runs

### 1) Realistic Demo (balanced)

```bash
./bin/spanforge \
  --format otlp-http \
  --output otlp \
  --otlp-endpoint http://localhost:4318 \
  --profile web \
  --services 8 \
  --routes 8 \
  --variety medium \
  --errors 0.5% \
  --retries 1% \
  --db-heavy 20% \
  --cache-hit-rate 85% \
  --rate 100 \
  --rate-unit traces \
  --rate-interval 1s \
  --duration 5m
```

### 2) Perf/Load Baseline (low cardinality)

```bash
./bin/spanforge \
  --format otlp-http \
  --output otlp \
  --otlp-endpoint http://localhost:4318 \
  --profile grpc \
  --services 6 \
  --routes 4 \
  --variety low \
  --errors 0.1% \
  --retries 0.2% \
  --high-cardinality=false \
  --rate 3000 \
  --rate-unit spans \
  --rate-interval 1s \
  --duration 2m
```

For pure generation benchmarking with sink disabled:

```bash
./bin/spanforge \
  --output noop \
  --format otlp-http \
  --rate 10000 \
  --rate-unit spans \
  --rate-interval 1s \
  --duration 30s
```

This prints a benchmark summary with emitted traces/spans and throughput.

You can also write a structured JSON report:

```bash
./bin/spanforge --output noop --format otlp-http --duration 15s --report-file ./out/report.json
```

### 3) High Variety Stress (demo richness)

```bash
./bin/spanforge \
  --format otlp-http \
  --output otlp \
  --otlp-endpoint http://localhost:4318 \
  --profile queue \
  --services 12 \
  --routes 24 \
  --variety high \
  --high-cardinality \
  --errors 2% \
  --retries 5% \
  --db-heavy 35% \
  --cache-hit-rate 70% \
  --rate 250 \
  --rate-unit traces \
  --rate-interval 1s \
  --duration 3m
```

## Docker Quickstart (Tempo)

1. Build the image:

```bash
docker build -t spanforge:dev .
```

2. Verify CLI in container:

```bash
docker run --rm spanforge:dev --version
```

3. Start collector + tempo + grafana + spanforge:

```bash
docker compose -f examples/docker-compose/tempo/docker-compose.yml up --build
```

4. Open Grafana at `http://localhost:${GRAFANA_PORT:-3000}`, go to Explore, select `Tempo`, and search traces.

5. Admin endpoints are available while spanforge is running.
   In Docker/container setups, use `--http-listen 0.0.0.0:8080` so the endpoint is reachable outside the container.
   Keep `127.0.0.1:8080` for host-only/local runs to avoid unnecessary exposure.

```bash
curl -s http://127.0.0.1:8080/healthz
curl -s http://127.0.0.1:8080/stats
```

6. Run spanforge manually from host against the collector:

```bash
./bin/spanforge \
  --format otlp-http \
  --output otlp \
  --otlp-endpoint http://localhost:4318 \
  --rate 100 \
  --rate-unit spans \
  --rate-interval 1s \
  --duration 0s \
  --http-listen 127.0.0.1:8080
```

For containerized runs (for example Docker Compose), use:

```bash
--http-listen 0.0.0.0:8080
```

## Releases

- CI workflow: `.github/workflows/ci.yml` (build + test + vet).
- Tag-based release workflow: `.github/workflows/release.yml`.
- GoReleaser config: `.goreleaser.yaml` (binaries + multi-arch Docker images/manifests for GHCR).

Required GitHub setup for releases:

1. Ensure repository Actions permission allows package write.
2. Ensure `GITHUB_TOKEN` has `contents: write` and `packages: write` (configured in workflow).
3. Create a semver tag (example: `v1.0.0`) and push it.

Release flow:

1. `git tag vX.Y.Z`
2. `git push origin vX.Y.Z`
3. GitHub Actions runs `release.yml` and publishes binaries + multi-arch GHCR images.

## Transport Perf Harness

Run OTLP HTTP vs OTLP gRPC throughput comparison:

```bash
make bench-transport
```

This runs `TestTransportPerfComparison` with `SPANFORGE_TRANSPORT_PERF=1` and logs spans/sec and grpc/http ratio.

## Docker Smoke CI

Workflow `.github/workflows/docker-smoke.yml` starts the Tempo and Jaeger compose examples, then checks:

- spanforge emits spans (`/stats`)
- collector exports spans (`otelcol_exporter_sent_spans` metric)
