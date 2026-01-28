# Usage Guide

This document provides instructions for building, installing, and using the Gardener Fluent Bit OTLP Output Plugin.

## Building

### Prerequisites

- Go 1.23+
- gcc (for cgo, required by Fluent Bit plugin interface)
- make

### Build the Plugin

```bash
make plugin
```

This builds the plugin as a shared library at `build/output_plugin.so`.

### Build the Event Logger

```bash
make event-logger
```

### Run Tests

```bash
make test
```

### Build Docker Images

```bash
make docker-images
```

## Installation

### With Fluent Bit Binary

If you have Fluent Bit installed in your `$PATH`:

```bash
fluent-bit -e /path/to/build/output_plugin.so -c fluent-bit.conf
```

### Using plugins.conf

You can also configure Fluent Bit to load the plugin automatically via `plugins.conf`:

```ini
[PLUGINS]
    Path /path/to/build/output_plugin.so
```

### Docker/Kubernetes

Mount the plugin as a volume and configure Fluent Bit to load it:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
data:
  plugins.conf: |
    [PLUGINS]
        Path /fluent-bit/plugins/output_plugin.so
  
  fluent-bit.conf: |
    [Service]
        Flush        1
        Daemon       off
        Log_Level    info
        Parsers_File parsers.conf
        Plugins_File plugins.conf
    
    [Input]
        Name              tail
        Path              /var/log/containers/*.log
        Parser            cri
        Tag               kubernetes.*
        Refresh_Interval  5
        Mem_Buf_Limit     5MB
        Skip_Long_Lines   On
    
    [Output]
        Name gardener
        Match kubernetes.*
        SeedType OTLPGRPC
        Endpoint victorialogs.logging.svc:4317
        LogLevel info
```

## Basic Usage

### Simple Configuration

Minimal configuration to send logs to a VictoriaLogs backend:

```ini
[Service]
    Flush        1
    Daemon       off
    Log_Level    info
    HTTP_Server  On
    HTTP_Listen  0.0.0.0
    HTTP_Port    2020

[Input]
    Name              tail
    Path              /var/log/containers/*.log
    Parser            cri
    Tag               kubernetes.*
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On

[Output]
    Name gardener
    Match kubernetes.*
    SeedType OTLPGRPC
    Endpoint victorialogs.logging.svc:4317
    LogLevel info
```

### Multiple Plugin Instances

You can run multiple plugin instances in the same Fluent Bit process to route logs to different backends:

```ini
# Route Kubernetes logs to Seed cluster
[Output]
    Name gardener
    Match kubernetes.*
    SeedType OTLPGRPC
    Endpoint victorialogs-seed.logging.svc:4317
    DQueDir /fluent-bit/buffers/kubernetes
    DQueName kubernetes-logs

# Route systemd logs to different endpoint
[Output]
    Name gardener
    Match systemd.*
    SeedType OTLPHTTP
    Endpoint https://victorialogs-systemd.logging.svc/insert/opentelemetry/v1/logs
    DQueDir /fluent-bit/buffers/systemd
    DQueName systemd-logs
```

## Running the Plugin

### Local Development

```bash
# Build the plugin
make plugin

# Run with Fluent Bit
fluent-bit -e ./build/output_plugin.so -c examples/fluent-bit.conf
```

### Debug Mode

Enable debug logging and profiling:

```ini
[Output]
    Name gardener
    Match *
    SeedType stdout
    LogLevel debug
    Pprof true
```

Then access:
- Logs: stdout
- Metrics: `http://localhost:2021/metrics`
- Health: `http://localhost:2021/healthz`
- Profiling: `http://localhost:2021/debug/pprof/`

### Production Deployment

For production, use the Docker images provided by the project:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
  namespace: logging
spec:
  selector:
    matchLabels:
      app: fluent-bit
  template:
    metadata:
      labels:
        app: fluent-bit
    spec:
      serviceAccountName: fluent-bit
      containers:
      - name: fluent-bit
        image: europe-docker.pkg.dev/gardener-project/releases/fluent-bit-output:latest
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 100Mi
        volumeMounts:
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        - name: fluent-bit-config
          mountPath: /fluent-bit/etc/
        - name: buffer
          mountPath: /fluent-bit/buffers
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
      - name: fluent-bit-config
        configMap:
          name: fluent-bit-config
      - name: buffer
        emptyDir: {}
```

## Common Use Cases

### Gardener Seed Cluster

Send logs from Seed cluster to central VictoriaLogs:

```ini
[Output]
    Name gardener
    Match *
    SeedType OTLPGRPC
    Endpoint victorialogs.garden.svc:4317
    
    # TLS configuration
    TLSCertFile /etc/ssl/fluent-bit/tls.crt
    TLSKeyFile /etc/ssl/fluent-bit/tls.key
    TLSCAFile /etc/ssl/fluent-bit/ca.crt
    
    # Buffering
    DQueDir /fluent-bit/buffers
    DQueName seed-logs
    
    # Metadata
    HostnameValue ${SEED_NAME}
    Origin seed
```

### Gardener Shoot Control Plane

Route logs from Shoot control planes to both Seed and Shoot clusters:

```ini
[Output]
    Name gardener
    Match kubernetes.*
    
    # Default Seed client
    SeedType OTLPGRPC
    Endpoint victorialogs-seed.logging.svc:4317
    
    # Dynamic Shoot routing
    ShootType OTLPGRPC
    DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
    DynamicHostPrefix victorialogs.
    DynamicHostSuffix .svc.cluster.local:4317
    DynamicHostRegex ^shoot--
    
    # State-based routing
    SendLogsToMainClusterWhenIsInReadyState true
    SendLogsToMainClusterWhenIsInHibernatingState false
    SendLogsToDefaultClientWhenClusterIsInHibernatingState true
    
    # Buffering per client
    DQueDir /fluent-bit/buffers/shoot-cp
    DQueName shoot-cp-logs
```

### Multi-Tenant Setup

Route logs to different tenants based on namespace:

```ini
[Output]
    Name gardener
    Match kubernetes.*
    
    # HTTP client for multi-tenancy
    SeedType OTLPHTTP
    Endpoint https://victorialogs.example.com/insert/opentelemetry/v1/logs
    
    # Extract tenant from namespace and add as header
    DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
    Headers X-Scope-OrgID ${NAMESPACE}
    
    # Authentication
    TLSCertFile /etc/ssl/fluent-bit/tls.crt
    TLSKeyFile /etc/ssl/fluent-bit/tls.key
```

## Verification

### Check Plugin Status

```bash
# Check Fluent Bit logs
kubectl logs -n logging daemonset/fluent-bit

# Check metrics
curl http://localhost:2021/metrics

# Check health
curl http://localhost:2021/healthz
```

### Verify Logs in Backend

```bash
# VictoriaLogs query
curl -G 'http://victorialogs:9428/select/logsql/query' \
  --data-urlencode 'query={origin="seed"} | limit 10'

# Check OTLP endpoint
grpcurl -plaintext victorialogs:4317 list
```

### Debug Issues

```bash
# Enable debug logging
# Set LogLevel debug in config

# Check queue status
curl http://localhost:2021/metrics | grep dque_queue_size

# Profile CPU usage
go tool pprof http://localhost:2021/debug/pprof/profile

# Check goroutines
curl http://localhost:2021/debug/pprof/goroutine?debug=2
```

## Performance Tuning

### High-Volume Environments

```ini
[Output]
    Name gardener
    Match *
    SeedType OTLPGRPC
    Endpoint victorialogs.logging.svc:4317
    
    # Larger batches
    DQueBatchProcessorMaxBatchSize 512
    DQueBatchProcessorMaxQueueSize 2048
    
    # Compression
    Compression 1
    
    # Faster flushing
    DQueBatchProcessorExportInterval 500ms
    
    # Buffering
    DQueDir /fluent-bit/buffers
    DQueSegmentSize 1000
```

### Low-Latency Requirements

For low-latency requirements with disk persistence:

```ini
[Output]
    Name gardener
    Match *
    SeedType OTLPGRPC
    Endpoint victorialogs.logging.svc:4317
    
    # Smaller batches, faster export
    DQueBatchProcessorMaxBatchSize 64
    DQueBatchProcessorExportInterval 100ms
    
    # No compression
    Compression 0
```

### Ultra Low-Latency with SDK BatchProcessor

For scenarios where minimal latency is critical and disk persistence is not required (e.g., ephemeral workloads or when using upstream buffering), use the OTEL SDK BatchProcessor:

```ini
[Output]
    Name gardener
    Match *
    SeedType OTLPGRPC
    Endpoint victorialogs.logging.svc:4317
    
    # Use OTEL SDK BatchProcessor (in-memory, no disk I/O overhead)
    UseSDKBatchProcessor true
    SDKBatchMaxQueueSize 1024
    SDKBatchExportTimeout 10s
    SDKBatchExportInterval 100ms
    SDKBatchExportMaxBatchSize 128
    
    # No compression for lowest latency
    Compression 0
```

> **Note:** SDK BatchProcessor stores logs in memory only. Logs will be lost if the process crashes or restarts before export. Use this option only when low latency is more important than durability.

### Memory-Constrained Environments

```ini
[Output]
    Name gardener
    Match *
    SeedType OTLPGRPC
    Endpoint victorialogs.logging.svc:4317
    
    # Smaller queue
    DQueBatchProcessorMaxQueueSize 256
    DQueBatchProcessorMaxBatchSize 128
    
    # Smaller segments
    DQueSegmentSize 250
```

## Next Steps

- Review [Configuration Guide](configuration.md) for detailed options
- See [Troubleshooting Guide](troubleshooting.md) for common issues
- Check [Architecture](architecture.md) for design details

