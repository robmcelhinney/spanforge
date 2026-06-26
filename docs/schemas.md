# Stable Schemas

spanforge emits two JSON documents that users can script against: run reports and validation results. These schemas are stable for release use. New fields may be added in minor releases; existing fields should not be removed or change type without a major release.

## Run Report JSON

Produced by `--report-file`.

```json
{
  "started_at": "2026-06-26T22:00:00Z",
  "finished_at": "2026-06-26T22:00:30Z",
  "duration_seconds": 30.0,
  "run_id": "sf_seed_1",
  "profile": "payment-system",
  "format": "otlp-http",
  "output": "otlp",
  "emitted_traces": 100,
  "emitted_spans": 1200,
  "traces_per_second": 3.33,
  "spans_per_second": 40.0,
  "services": ["edge-gateway", "checkout-api"],
  "sample_trace_ids": ["1549771af576db8076a..."],
  "phases": [
    {
      "name": "warmup",
      "traces_sent": 10,
      "spans_sent": 120
    }
  ]
}
```

Stable fields:

| Field | Type | Notes |
| --- | --- | --- |
| `started_at` | string | RFC3339 timestamp. |
| `finished_at` | string | RFC3339 timestamp. |
| `duration_seconds` | number | Wall-clock run duration. |
| `run_id` | string | Stable run identifier also emitted as `spanforge.run_id`. |
| `profile` | string | Stable profile name. |
| `format` | string | Output format selected for the run. |
| `output` | string | Sink selected for the run. |
| `emitted_traces` | number | Traces emitted by spanforge before backend ingestion effects. |
| `emitted_spans` | number | Spans emitted by spanforge before backend ingestion effects. |
| `traces_per_second` | number | Local emission rate. |
| `spans_per_second` | number | Local emission rate. |
| `services` | array of strings | Services observed in generated traces. |
| `sample_trace_ids` | array of strings | Trace IDs suitable for backend validation. |
| `phases` | array | Present when `--load` or `--phase-file` is used. |

## Validation Result JSON

Produced by `spanforge validate tempo --output json` and `spanforge validate jaeger --output json`.

```json
{
  "status": "pass",
  "backend": "tempo",
  "endpoint": "http://localhost:3200",
  "checks": [
    {
      "name": "sample_traces",
      "status": "pass",
      "message": "found all 3 sampled traces"
    }
  ]
}
```

Stable fields:

| Field | Type | Notes |
| --- | --- | --- |
| `status` | string | Overall `pass`, `warn`, or `fail`. |
| `backend` | string | `tempo` or `jaeger`. |
| `endpoint` | string | Backend query endpoint used. |
| `checks` | array | Individual validation checks. |
| `checks[].name` | string | Stable check identifier. |
| `checks[].status` | string | `pass`, `warn`, or `fail`. |
| `checks[].message` | string | Human-readable detail; wording may change in minor releases. |

Stable check names:

- `sample_traces`
- `run_id`
- `services`
- `phase_labels`
- `error_spans`
- `high_latency_spans`

## Migration Policy

- Additive fields are allowed in minor releases.
- Consumers should ignore unknown fields.
- Consumers should treat `message` as display text, not a machine contract.
- Removing fields, changing field types, or renaming stable check names requires a major release.
