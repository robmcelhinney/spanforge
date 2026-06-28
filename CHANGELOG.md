# Changelog

This file records notable changes to spanforge. Release links point to the matching GitHub comparison or tag.

## v0.2.0

Released on 27 June 2026.

### Added

- `payment-system` and `api-gateway` profiles with stable run metadata
- load phases and built-in load patterns
- Tempo and Grafana demo with a prepared dashboard
- Tempo and Jaeger backend validation
- weird telemetry modes for testing unusual but valid traces
- invalid telemetry modes for testing backend rejection
- stable report and validation schemas
- backend compatibility and contribution guidance

[View changes from v0.1.2 to v0.2.0](https://github.com/robmcelhinney/spanforge/compare/v0.1.2...v0.2.0).

## v0.1.2

Released on 26 February 2026.

### Added

- unlimited runs with `--duration 0s`
- debug logs for trace emission and sink requests

[View changes from v0.1.1 to v0.1.2](https://github.com/robmcelhinney/spanforge/compare/v0.1.1...v0.1.2).

## v0.1.1

Released on 16 February 2026.

### Changed

- updated the project to Go 1.26
- updated releases to GoReleaser 2 and `dockers_v2`
- made Docker smoke tests wait for services more reliably

[View changes from v0.1.0 to v0.1.1](https://github.com/robmcelhinney/spanforge/compare/v0.1.0...v0.1.1).

## v0.1.0

Tagged on 16 February 2026. This version was not published as a GitHub release.

### Added

- trace generation with deterministic seeds and rate control
- OTLP HTTP, OTLP gRPC, Zipkin v2 JSON, JSONL and pretty output
- standard output, file, OTLP, Zipkin and noop sinks
- YAML files, environment variables and command-line options
- run reports, health checks and statistics
- Docker, Tempo and Jaeger examples
- build, test and release workflows

[View the v0.1.0 tag](https://github.com/robmcelhinney/spanforge/tree/v0.1.0).
