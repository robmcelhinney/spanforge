# spanforge Tempo/Grafana Demo

This stack starts:

- OpenTelemetry Collector
- Tempo
- Grafana with a pre-provisioned Tempo datasource
- spanforge running the checkout brownout phase workload

Run:

```bash
docker compose up --build
```

Open Grafana:

```text
http://localhost:3000
```

The home dashboard is `Spanforge Overview`. It uses the Tempo datasource and TraceQL searches for recent traces, error traces, OK traces, and duration thresholds.

![Spanforge Tempo/Grafana dashboard](../../../docs/assets/spanforge-overview-dashboard.png)

The spanforge container restarts after each finite phase-file run so the dashboard keeps receiving fresh synthetic traffic during demos.

Useful endpoints:

- Tempo: `http://localhost:3200`
- OTel Collector OTLP HTTP: `http://localhost:4318`
- OTel Collector metrics: `http://localhost:18888/metrics`
- spanforge admin: `http://localhost:8080/stats`
