# Monitoring and Metrics

This document describes the metrics, health checks, and observability features of the Gardener Fluent Bit OTLP Output Plugin.

## Metrics Endpoints

The plugin exposes multiple HTTP endpoints for monitoring:

| Endpoint | Port | Description |
|----------|------|-------------|
| `/metrics` | 2021 | Prometheus metrics |
| `/healthz` | 2021 | Health check endpoint |
| `/debug/pprof/*` | 2021 | Profiling endpoints (when enabled) |

## Prometheus Metrics

### Plugin Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `flb_gardener_errors_total` | Counter | `error_type` | Total errors by type |
| `flb_gardener_records_total` | Counter | `client_type` | Total records processed |
| `flb_gardener_clients_total` | Gauge | - | Total active clients |

**Example:**
```promql
# Error rate by type
rate(flb_gardener_errors_total[5m])

# Records per second
rate(flb_gardener_records_total[1m])

# Active clients
flb_gardener_clients_total
```

### DQue (Queue) Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dque_queue_size` | Gauge | `endpoint` | Current queue size (records) |
| `dque_enqueued_total` | Counter | `endpoint` | Total records enqueued |
| `dque_dequeued_total` | Counter | `endpoint` | Total records dequeued |
| `dque_dropped_total` | Counter | `endpoint` | Total records dropped (queue full) |
| `dque_export_duration_seconds` | Histogram | `endpoint` | Export operation duration |
| `dque_export_errors_total` | Counter | `endpoint`, `error_type` | Total export errors |

**Example Queries:**

```promql
# Queue depth
dque_queue_size

# Enqueue rate
rate(dque_enqueued_total[5m])

# Drop rate (should be 0)
rate(dque_dropped_total[5m])

# Export latency (95th percentile)
histogram_quantile(0.95, rate(dque_export_duration_seconds_bucket[5m]))

# Export error rate
rate(dque_export_errors_total[5m])
```

### OTLP Client Metrics

The OTLP SDK automatically exports additional metrics:

| Metric | Description |
|--------|-------------|
| `otelcol_exporter_sent_log_records` | Number of log records successfully sent |
| `otelcol_exporter_send_failed_log_records` | Number of log records failed to send |
| `otelcol_exporter_queue_size` | Current queue size |
| `otelcol_exporter_queue_capacity` | Maximum queue capacity |

## Health Checks

### Health Endpoint

```bash
curl http://localhost:2021/healthz
```

**Responses:**

- `200 OK`: Plugin is healthy
- `503 Service Unavailable`: Plugin is unhealthy

### Health Check Logic

The plugin is considered healthy when:
- All required services are running
- No critical errors in recent history
- At least one client is operational

## Alerting Rules

### Recommended Prometheus Alerts

```yaml
groups:
- name: fluent-bit-gardener-plugin
  interval: 30s
  rules:
  
  # Queue growing continuously
  - alert: FluentBitQueueGrowing
    expr: |
      delta(dque_queue_size[5m]) > 100
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Fluent Bit queue growing on {{ $labels.instance }}"
      description: "Queue size increased by {{ $value }} in 5 minutes. Backend may be slow or unavailable."
  
  # Queue near capacity
  - alert: FluentBitQueueNearCapacity
    expr: |
      dque_queue_size / on() group_left() flb_gardener_clients_total > 400
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Fluent Bit queue near capacity on {{ $labels.instance }}"
      description: "Queue size is {{ $value }} (max 512). Logs may be dropped soon."
  
  # Logs being dropped
  - alert: FluentBitLogsDropped
    expr: |
      rate(dque_dropped_total[5m]) > 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Fluent Bit dropping logs on {{ $labels.instance }}"
      description: "{{ $value }} logs/sec are being dropped. Queue is full."
  
  # High export error rate
  - alert: FluentBitHighErrorRate
    expr: |
      rate(dque_export_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High export error rate on {{ $labels.instance }}"
      description: "{{ $value }} export errors/sec. Check backend connectivity."
  
  # High export latency
  - alert: FluentBitHighExportLatency
    expr: |
      histogram_quantile(0.95,
        rate(dque_export_duration_seconds_bucket[5m])
      ) > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High export latency on {{ $labels.instance }}"
      description: "95th percentile export latency is {{ $value }}s. Backend may be slow."
  
  # Plugin not healthy
  - alert: FluentBitPluginUnhealthy
    expr: |
      up{job="fluent-bit"} == 0
      or
      probe_success{job="fluent-bit-healthz"} == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Fluent Bit plugin unhealthy on {{ $labels.instance }}"
      description: "Plugin is not responding to health checks."
  
  # No logs being sent
  - alert: FluentBitNoLogsSent
    expr: |
      rate(dque_dequeued_total[5m]) == 0
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Fluent Bit not sending logs on {{ $labels.instance }}"
      description: "No logs exported in 10 minutes. Check configuration and backend."
```

## Grafana Dashboards

### Overview Dashboard

Example dashboard JSON snippet:

```json
{
  "dashboard": {
    "title": "Fluent Bit Gardener Plugin",
    "panels": [
      {
        "title": "Queue Size",
        "targets": [
          {
            "expr": "dque_queue_size",
            "legendFormat": "{{ endpoint }}"
          }
        ]
      },
      {
        "title": "Enqueue Rate",
        "targets": [
          {
            "expr": "rate(dque_enqueued_total[1m])",
            "legendFormat": "{{ endpoint }}"
          }
        ]
      },
      {
        "title": "Export Latency (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(dque_export_duration_seconds_bucket[5m]))",
            "legendFormat": "{{ endpoint }}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(dque_export_errors_total[5m])",
            "legendFormat": "{{ endpoint }} - {{ error_type }}"
          }
        ]
      }
    ]
  }
}
```

### Key Metrics to Monitor

1. **Throughput**:
   - Enqueue rate: `rate(dque_enqueued_total[1m])`
   - Dequeue rate: `rate(dque_dequeued_total[1m])`
   - Records processed: `rate(flb_gardener_records_total[1m])`

2. **Latency**:
   - Export duration p50: `histogram_quantile(0.5, rate(dque_export_duration_seconds_bucket[5m]))`
   - Export duration p95: `histogram_quantile(0.95, rate(dque_export_duration_seconds_bucket[5m]))`
   - Export duration p99: `histogram_quantile(0.99, rate(dque_export_duration_seconds_bucket[5m]))`

3. **Errors**:
   - Error rate: `rate(flb_gardener_errors_total[5m])`
   - Export errors: `rate(dque_export_errors_total[5m])`
   - Drop rate: `rate(dque_dropped_total[5m])`

4. **Queue Health**:
   - Queue size: `dque_queue_size`
   - Queue utilization: `dque_queue_size / 512 * 100`
   - Queue growth: `delta(dque_queue_size[5m])`

## Profiling (Debug)

### Enabling Profiling

Enable pprof in configuration:

```ini
[Output]
    Name gardener
    Match *
    Pprof true
```

### Available Profiles

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/` | Index of available profiles |
| `/debug/pprof/goroutine` | Stack traces of all goroutines |
| `/debug/pprof/heap` | Heap memory allocations |
| `/debug/pprof/allocs` | All past memory allocations |
| `/debug/pprof/threadcreate` | Stack traces of thread creation |
| `/debug/pprof/block` | Stack traces of blocking operations |
| `/debug/pprof/mutex` | Stack traces of mutex contention |
| `/debug/pprof/profile` | CPU profile (30s sample) |
| `/debug/pprof/trace` | Execution trace (1s sample) |

### Using Profiles

#### CPU Profile

```bash
# Collect 30-second CPU profile
go tool pprof http://localhost:2021/debug/pprof/profile

# In pprof interactive mode:
(pprof) top10
(pprof) list functionName
(pprof) web
```

#### Heap Profile

```bash
# Analyze heap usage
go tool pprof http://localhost:2021/debug/pprof/heap

# Show top memory consumers
(pprof) top10

# Show allocations by function
(pprof) list functionName
```

#### Goroutine Profile

```bash
# Check goroutine count
curl http://localhost:2021/debug/pprof/goroutine?debug=1

# Analyze with pprof
go tool pprof http://localhost:2021/debug/pprof/goroutine
```

#### Trace Analysis

```bash
# Collect execution trace
curl http://localhost:2021/debug/pprof/trace?seconds=5 > trace.out

# Analyze trace
go tool trace trace.out
```

## Log Analysis

### Plugin Logs

The plugin logs to Fluent Bit's output. Key log messages:

**Startup:**
```
[info] Starting fluent-bit-gardener-output-plugin version=v1.0.0 revision=abc123
[info] [flb-go] output plugin initialized id=xyz count=1
```

**Normal Operation:**
```
[debug] [flb-go] sending batch records=256 endpoint=victorialogs:4317
[debug] [flb-go] batch exported duration=123ms
```

**Errors:**
```
[error] [flb-go] failed to export batch error="connection refused"
[error] [flb-go] queue full, dropping records count=10
```

### Log Levels

Set appropriate log level:

```ini
# Production
LogLevel info

# Troubleshooting
LogLevel debug

# Silent (errors only)
LogLevel error
```

## Monitoring Best Practices

### What to Monitor

1. **Core Metrics**:
   - Queue size (should be stable and low)
   - Export latency (should be consistent)
   - Error rate (should be near zero)
   - Drop rate (should be zero)

2. **Resource Usage**:
   - Memory usage (pod/container metrics)
   - CPU usage (should be consistent)
   - Disk I/O (queue storage)
   - Network bandwidth

3. **Client Health**:
   - Active client count
   - Per-client queue sizes
   - Per-client error rates

### Alert Thresholds

Recommended thresholds:

```yaml
# Warning thresholds
queue_size_warning: 400  # 80% of default max (512)
export_latency_warning: 5s
error_rate_warning: 0.01  # 1%

# Critical thresholds
logs_dropped: > 0  # Any drops
export_latency_critical: 10s
error_rate_critical: 0.1  # 10%
health_check_failure: true
```

### SLOs (Service Level Objectives)

Example SLOs:

```yaml
# Availability
- metric: up
  target: 99.9%
  window: 30d

# Error rate
- metric: rate(dque_export_errors_total[5m]) / rate(dque_enqueued_total[5m])
  target: < 0.1%  # 99.9% success rate
  window: 30d

# Latency
- metric: histogram_quantile(0.95, rate(dque_export_duration_seconds_bucket[5m]))
  target: < 5s
  window: 30d

# No data loss
- metric: rate(dque_dropped_total[5m])
  target: 0
  window: 30d
```

## Troubleshooting with Metrics

### Queue Growing

```promql
# Check queue growth
delta(dque_queue_size[5m]) > 100

# Check export vs enqueue rate
rate(dque_enqueued_total[1m]) - rate(dque_dequeued_total[1m])
```

**Actions:**
- Increase export rate
- Check backend performance
- Enable compression
- Increase batch size

### High Latency

```promql
# Check p95 latency
histogram_quantile(0.95, rate(dque_export_duration_seconds_bucket[5m])) > 5
```

**Actions:**
- Check network latency to backend
- Review backend performance
- Reduce batch size
- Check for throttling

### Memory Issues

```promql
# Check memory usage (from container metrics)
container_memory_usage_bytes{pod=~"fluent-bit.*"} / container_spec_memory_limit_bytes{pod=~"fluent-bit.*"} > 0.8
```

**Actions:**
- Reduce queue size
- Reduce batch size
- Check for goroutine leaks (pprof)

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)
- [Go pprof Guide](https://go.dev/blog/pprof)
- [Troubleshooting Guide](troubleshooting.md)

