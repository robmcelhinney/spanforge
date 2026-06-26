# Compatibility Matrix

This matrix documents the backend routes spanforge treats as supported for release-quality use. Version entries are intentionally conservative: they reflect the compose examples and protocol paths this repository tests or documents.

| Backend | Supported path | Tested example | Expected result | Notes |
| --- | --- | --- | --- | --- |
| OpenTelemetry Collector | OTLP HTTP on `4318`, OTLP gRPC on `4317` | Collector contrib image in `examples/docker-compose/*` | Collector accepts generated spans and exports them to downstream backends | Preferred integration path for Tempo and Jaeger. |
| Tempo | `spanforge -> Collector -> Tempo` over OTLP | `examples/docker-compose/tempo` and `examples/docker-compose/tempo-grafana` | Traces searchable by sample trace ID and visible in Grafana Tempo datasource | TraceQL metrics are not required by the bundled dashboard. |
| Jaeger | `spanforge -> Collector -> Jaeger` over OTLP | `examples/docker-compose/jaeger` | Traces searchable in Jaeger UI and query API | Direct Jaeger thrift output is not part of the stable surface. |
| Zipkin | Direct Zipkin v2 JSON POST to `/api/v2/spans` | Zipkin encoder and sink tests | Zipkin-compatible spans are accepted by Zipkin API | Zipkin validation command is not implemented; validate via backend UI/API. |

## Stable Commands

These command groups are part of the stable CLI surface for the next release:

- `spanforge`
- `spanforge profiles list`
- `spanforge profiles show <name>`
- `spanforge validate tempo`
- `spanforge validate jaeger`

Stable profile names:

- `web`
- `grpc`
- `queue`
- `batch`
- `payment-system`
- `api-gateway`

## Compatibility Policy

- Patch releases should not remove profile names, report fields, validation fields, or command flags.
- Minor releases may add fields, checks, profiles, and backend examples.
- Breaking changes require a major release note and a migration note in `docs/schemas.md`.
- Backend validation avoids exact trace-count promises. It validates sampled trace IDs, run IDs, services, phase labels, error spans, and high-latency spans because backend ingestion, retention, and query windows vary.
