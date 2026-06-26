# Contributing

Thanks for working on spanforge. The project is a Go CLI for generating synthetic traces, so changes should keep the command line predictable and backend behavior easy to verify.

## Development

Run the full test suite before sending changes:

```bash
go test ./...
```

Useful local checks:

```bash
go run ./cmd/spanforge --help
go run ./cmd/spanforge profiles list
go run ./cmd/spanforge validate tempo --help
```

Docker examples:

```bash
docker compose -f examples/docker-compose/tempo/docker-compose.yml up --build
docker compose -f examples/docker-compose/jaeger/docker-compose.yml up --build
docker compose -f examples/docker-compose/tempo-grafana/docker-compose.yml up --build
```

## Compatibility Expectations

- Keep stable profile names working: `web`, `grpc`, `queue`, `batch`, `payment-system`, `api-gateway`.
- Keep report JSON and validation JSON compatible with `docs/schemas.md`.
- Prefer additive fields and flags over breaking changes.
- Avoid exact backend trace-count assertions unless a backend API makes them reliable.
- Use the Collector route for backend support unless there is a strong reason to add a direct sink.

## Pull Requests

Good pull requests include:

- A focused problem statement.
- Tests for behavior changes.
- Docs updates for CLI flags, schema fields, examples, or compatibility changes.
- Notes about backend versions or APIs used when touching Tempo, Jaeger, Zipkin, or Collector behavior.

## Release Changes

Release-facing changes should update:

- `README.md` for quick-start behavior.
- `docs/further-info.md` for operational runbooks.
- `docs/compatibility.md` for backend support changes.
- `docs/schemas.md` for report or validation JSON changes.
- `spanforge_action_checklist.md` when completing roadmap steps.
