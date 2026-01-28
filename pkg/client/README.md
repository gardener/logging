# Client Package

The `client` package provides multiple implementations of the `OutputClient` interface for sending logs from Fluent Bit to various backends. It supports OpenTelemetry Protocol (OTLP) over gRPC and HTTP, as well as stdout and no-op clients for testing and debugging.

## Table of Contents

- [Overview](#overview)
- [Client Types](#client-types)
  - [OTLP gRPC Client](#otlp-grpc-client)
  - [OTLP HTTP Client](#otlp-http-client)
  - [Stdout Client](#stdout-client)
  - [Noop Client](#noop-client)
- [Target Types](#target-types)
- [Configuration](#configuration)
  - [OTLP Configuration](#otlp-configuration)
  - [DQue Configuration](#dque-configuration)
  - [Batch Processor Configuration](#batch-processor-configuration)
  - [TLS Configuration](#tls-configuration)
  - [Retry Configuration](#retry-configuration)
  - [Throttle Configuration](#throttle-configuration)
- [Usage](#usage)
  - [Creating a Client](#creating-a-client)
  - [Client Options](#client-options)
  - [Handling Logs](#handling-logs)
  - [Shutting Down](#shutting-down)
- [Architecture](#architecture)
  - [DQue Batch Processor](#dque-batch-processor)
  - [Metrics](#metrics)
- [Examples](#examples)

## Overview

The client package abstracts the complexity of sending logs to different backends. All clients implement the `OutputClient` interface, which provides a consistent API regardless of the underlying transport mechanism.

The package supports:
- **Multiple protocols**: OTLP over gRPC and HTTP
- **Persistent buffering**: Using dque (disk-based queue) for reliability
- **Batch processing**: Efficient log batching with configurable limits
- **Retry logic**: Configurable exponential backoff for failed exports
- **Rate limiting**: Optional throttling to prevent overwhelming backends
- **TLS/mTLS**: Full TLS configuration support
- **Metrics**: Prometheus metrics for monitoring client behavior
- **Target-based routing**: Separate configurations for Seed and Shoot clusters

## Client Types

### OTLP gRPC Client

The OTLP gRPC client (`OTLPGRPCClient`) sends logs using the OpenTelemetry Protocol over gRPC. This is the recommended production client for high-throughput, low-latency log shipping.

**Features:**
- Bi-directional streaming support
- Efficient binary protocol (Protobuf)
- Built-in compression (gzip)
- Connection multiplexing
- Persistent buffering with dque
- Configurable batch processing
- Retry with exponential backoff
- Optional rate limiting
- TLS/mTLS support
- gRPC metrics instrumentation

**Use cases:**
- Production environments
- High-volume log shipping
- Low-latency requirements
- When backend supports gRPC

**Configuration type:** `otlp_grpc` (string) or `types.OTLPGRPC` (enum)

### OTLP HTTP Client

The OTLP HTTP client (`OTLPHTTPClient`) sends logs using the OpenTelemetry Protocol over HTTP/1.1 or HTTP/2.

**Features:**
- Standard HTTP protocol
- JSON or Protobuf encoding
- Compression support (gzip)
- Persistent buffering with dque
- Configurable batch processing
- Retry with exponential backoff
- Optional rate limiting
- TLS support
- Works through HTTP proxies

**Use cases:**
- When gRPC is not available or blocked by firewalls
- HTTP proxy environments
- Debugging (easier to inspect with standard tools)
- When backend only supports HTTP

**Configuration type:** `otlp_http` (string) or `types.OTLPHTTP` (enum)

### Stdout Client

The Stdout client (`StdoutClient`) writes all log entries to standard output in JSON format.

**Features:**
- Simple JSON output
- No external dependencies
- Minimal overhead
- Useful for debugging
- Metrics tracking

**Use cases:**
- Development and debugging
- Testing log processing pipeline
- Integration with stdout-based log collectors
- Troubleshooting without backend connectivity

**Configuration type:** `stdout` (string) or `types.STDOUT` (enum)

**Output format:**
```json
{
  "timestamp": "2025-12-22T10:30:45.123456Z",
  "record": {
    "message": "Application log message",
    "level": "info",
    "kubernetes": {...}
  }
}
```

### Noop Client

The Noop client (`NoopClient`) discards all log entries without processing them.

**Features:**
- Zero overhead
- Discards all logs
- Increments dropped logs metrics
- Useful for testing

**Use cases:**
- Performance testing (measure overhead without I/O)
- Disabling log output temporarily
- Testing metrics collection
- Benchmarking

**Configuration type:** `noop` (string) or `types.NOOP` (enum)

## Target Types

The client package supports two target types that determine which backend configuration to use:

### Seed Target

The Seed target (`client.Seed`) is used for logs originating from the Gardener Seed cluster. The client uses the `SeedType` configuration from `PluginConfig` to determine which client implementation to create.

**Usage:**
```go
client, err := client.NewClient(ctx, cfg, client.WithTarget(client.Seed))
```

### Shoot Target

The Shoot target (`client.Shoot`) is used for logs originating from the Gardener Shoot clusters. The client uses the `ShootType` configuration from `PluginConfig` to determine which client implementation to create.

**Usage:**
```go
client, err := client.NewClient(ctx, cfg, client.WithTarget(client.Shoot))
```

## Configuration

Configuration is managed through the `config.Config` struct, which contains both plugin-level and OTLP-specific settings.

### OTLP Configuration

The `OTLPConfig` struct holds all OTLP-related configuration:

```go
type OTLPConfig struct {
    Endpoint    string            // Backend endpoint (e.g., "localhost:4317")
    Insecure    bool              // Skip TLS verification (not recommended for production)
    Compression int               // Compression level (0 = none, 1 = gzip)
    Timeout     time.Duration     // Request timeout
    Headers     map[string]string // Custom HTTP/gRPC headers (e.g., authentication)
    
    // Embedded configurations
    DQueConfig                     // Persistent queue settings
    
    // Batch processor settings
    DQueBatchProcessorMaxQueueSize     int
    DQueBatchProcessorMaxBatchSize     int
    DQueBatchProcessorExportTimeout    time.Duration
    DQueBatchProcessorExportInterval   time.Duration
    DQueBatchProcessorExportBufferSize int
    
    // Retry settings
    RetryEnabled         bool
    RetryInitialInterval time.Duration
    RetryMaxInterval     time.Duration
    RetryMaxElapsedTime  time.Duration
    
    // Throttle settings
    ThrottleEnabled        bool
    ThrottleRequestsPerSec int
    
    // TLS settings
    TLSCertFile           string
    TLSKeyFile            string
    TLSCAFile             string
    TLSServerName         string
    TLSInsecureSkipVerify bool
    TLSMinVersion         string
    TLSMaxVersion         string
}
```

**Default values:**
```go
Endpoint:                           "localhost:4317"
Insecure:                           false
Compression:                        0  // No compression
Timeout:                            30 * time.Second
RetryEnabled:                       true
RetryInitialInterval:               5 * time.Second
RetryMaxInterval:                   30 * time.Second
RetryMaxElapsedTime:                1 * time.Minute
ThrottleEnabled:                    false
ThrottleRequestsPerSec:             0  // No limit
DQueBatchProcessorMaxQueueSize:     512
DQueBatchProcessorMaxBatchSize:     256
DQueBatchProcessorExportTimeout:    30 * time.Second
DQueBatchProcessorExportInterval:   1 * time.Second
TLSMinVersion:                      "1.2"
```

### DQue Configuration

The `DQueConfig` struct configures the persistent disk-based queue:

```go
type DQueConfig struct {
    DQueDir         string // Directory for queue persistence
    DQueSegmentSize int    // Number of items per segment file
    DQueSync        bool   // Synchronous writes (slower but safer)
    DQueName        string // Queue name (for multiple queues)
}
```

**Default values:**
```go
DQueDir:         "/tmp/flb-storage"
DQueSegmentSize: 500
DQueSync:        false
DQueName:        "dque"
```

**Considerations:**
- **DQueDir**: Ensure sufficient disk space and proper permissions
- **DQueSegmentSize**: Larger values = fewer files, smaller values = faster recovery
- **DQueSync**: Enable for critical logs, disable for performance
- **DQueName**: Use unique names when running multiple instances

### Batch Processor Configuration

The batch processor groups logs into batches before sending to reduce overhead:

| Parameter | Description | Default | Tuning |
|-----------|-------------|---------|--------|
| `DQueBatchProcessorMaxQueueSize` | Maximum records in memory queue before dropping | 512 | Increase for high throughput, decrease to prevent OOM |
| `DQueBatchProcessorMaxBatchSize` | Maximum records per export batch | 256 | Increase for efficiency, decrease for lower latency |
| `DQueBatchProcessorExportTimeout` | Timeout for single export operation | 30s | Increase for slow backends |
| `DQueBatchProcessorExportInterval` | Time between periodic exports | 1s | Decrease for lower latency, increase for efficiency |

**Tuning guidelines:**
- High throughput: Increase `MaxBatchSize` and `ExportInterval`
- Low latency: Decrease `ExportInterval` and `MaxBatchSize`
- Memory constrained: Decrease `MaxQueueSize`
- Slow backend: Increase `ExportTimeout`

### TLS Configuration

TLS is configured through the `OTLPConfig` fields:

```go
cfg.OTLPConfig.TLSCertFile = "/path/to/client-cert.pem"  // Client certificate (for mTLS)
cfg.OTLPConfig.TLSKeyFile = "/path/to/client-key.pem"    // Client private key (for mTLS)
cfg.OTLPConfig.TLSCAFile = "/path/to/ca-cert.pem"        // CA certificate for server verification
cfg.OTLPConfig.TLSServerName = "example.com"             // Server name for SNI
cfg.OTLPConfig.TLSInsecureSkipVerify = false             // Don't skip verification (recommended)
cfg.OTLPConfig.TLSMinVersion = "1.2"                     // Minimum TLS version
cfg.OTLPConfig.TLSMaxVersion = "1.3"                     // Maximum TLS version (optional)
```

**Security best practices:**
- Always use TLS in production
- Never set `Insecure` or `TLSInsecureSkipVerify` to `true` in production
- Use TLS 1.2 or higher
- Implement mTLS for enhanced security
- Keep certificates rotated and up-to-date

### Retry Configuration

Retry configuration uses exponential backoff:

```go
cfg.OTLPConfig.RetryEnabled = true
cfg.OTLPConfig.RetryInitialInterval = 5 * time.Second   // First retry after 5s
cfg.OTLPConfig.RetryMaxInterval = 30 * time.Second      // Max wait between retries
cfg.OTLPConfig.RetryMaxElapsedTime = 1 * time.Minute    // Give up after 1 minute
```

**Retry sequence example:**
1. Initial request fails
2. Wait 5s, retry
3. Wait 10s, retry (doubled)
4. Wait 20s, retry (doubled)
5. Wait 30s, retry (capped at max)
6. Continue until 1 minute elapsed, then give up

### Throttle Configuration

Rate limiting prevents overwhelming the backend:

```go
cfg.OTLPConfig.ThrottleEnabled = true
cfg.OTLPConfig.ThrottleRequestsPerSec = 100  // Max 100 requests/second
```

**Behavior:**
- When enabled, client limits requests to specified rate
- Excess requests return `ErrThrottled` error
- Use `DroppedLogs` metrics to monitor throttled records
- Set `ThrottleRequestsPerSec = 0` for unlimited (when `ThrottleEnabled = false`)

## Usage

### Creating a Client

Use the `NewClient` function with functional options:

```go
import (
    "context"
    "github.com/gardener/logging/v1/pkg/client"
    "github.com/gardener/logging/v1/pkg/config"
    "github.com/go-logr/logr"
)

// Load configuration
cfg := config.Config{
    PluginConfig: config.PluginConfig{
        ShootType: "otlp_grpc",  // Client type for shoot clusters
        SeedType: "otlp_grpc",   // Client type for seed cluster
    },
    OTLPConfig: config.OTLPConfig{
        Endpoint: "otlp-collector.example.com:4317",
        Timeout:  30 * time.Second,
    },
}

// Create logger (use your preferred logging library)
logger := logr.Discard()  // Replace with actual logger

// Create client for shoot target
ctx := context.Background()
shootClient, err := client.NewClient(
    ctx,
    cfg,
    client.WithTarget(client.Shoot),
    client.WithLogger(logger),
)
if err != nil {
    // Handle error
}
defer shootClient.StopWait()
```

### Client Options

The `NewClient` function accepts functional options:

#### WithTarget

Specifies whether to use Seed or Shoot configuration:

```go
client.NewClient(ctx, cfg, client.WithTarget(client.Shoot))
client.NewClient(ctx, cfg, client.WithTarget(client.Seed))
```

#### WithLogger

Provides a logger for client operations:

```go
logger := logr.New(handler)  // Your logger implementation
client.NewClient(ctx, cfg, client.WithLogger(logger))
```

If no logger is provided, a no-op logger is used.

### Handling Logs

Once created, use the `Handle` method to send logs:

```go
entry := types.OutputEntry{
    Timestamp: time.Now(),
    Record: map[string]any{
        "message": "Application started",
        "level":   "info",
        "kubernetes": map[string]any{
            "namespace": "default",
            "pod":       "app-123",
        },
    },
}

err := shootClient.Handle(entry)
if err != nil {
    // Handle error (e.g., throttled, queue full, network error)
}
```

**Error handling:**
- `client.ErrThrottled`: Client is rate-limited
- `client.ErrQueueFull`: Internal queue is full
- `client.ErrProcessorClosed`: Client has been shut down
- Network errors: Backend unreachable or request failed

### Shutting Down

Proper shutdown ensures all buffered logs are sent:

#### Graceful Shutdown (Recommended)

```go
// Stop accepting new logs and wait for queue to drain
shootClient.StopWait()
```

This method:
1. Stops accepting new logs
2. Flushes all buffered logs to backend
3. Waits for in-flight exports to complete
4. Closes connections

#### Immediate Shutdown

```go
// Stop immediately without waiting
shootClient.Stop()
```

This method:
1. Stops accepting new logs immediately
2. Cancels in-flight operations
3. May lose buffered logs
4. Use only in emergency or testing

**Best practice:**
```go
defer shootClient.StopWait()  // Ensure graceful shutdown on exit
```

## Architecture

### DQue Batch Processor

The DQue Batch Processor is the core component for reliable log delivery:

```
┌─────────────────────────────────────────────────────────────────┐
│                       Fluent Bit Output Plugin                   │
└────────────────────────────────┬────────────────────────────────┘
                                 │ Handle(entry)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                         OutputClient                             │
│  (OTLPGRPCClient / OTLPHTTPClient / StdoutClient / NoopClient)  │
└────────────────────────────────┬────────────────────────────────┘
                                 │ OnEmit(record)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    DQue Batch Processor                          │
│                                                                   │
│  ┌────────────────────┐         ┌──────────────────────┐        │
│  │  Memory Queue      │────────▶│  DQue (Disk Queue)   │        │
│  │  (Circular Buffer) │         │  (Persistent Storage)│        │
│  └────────────────────┘         └──────────────────────┘        │
│            │                              │                      │
│            │ Batch (every 1s or 256 logs) │                      │
│            ▼                              │                      │
│  ┌────────────────────┐                  │                      │
│  │  Export Worker     │◀─────────────────┘                      │
│  │  (Goroutine)       │                                          │
│  └────────┬───────────┘                                          │
└───────────┼──────────────────────────────────────────────────────┘
            │ Export(batch)
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      OTLP Exporter                               │
│              (gRPC or HTTP with retry logic)                     │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                                 ▼
                        Backend (e.g., Vali)
```

**Key features:**
1. **Memory Queue**: Fast in-memory circular buffer for incoming logs
2. **Persistent Storage**: DQue writes logs to disk for durability
3. **Batch Processing**: Groups logs into efficient batches
4. **Export Worker**: Background goroutine handles exports
5. **Retry Logic**: Automatic retry with exponential backoff
6. **Metrics**: Comprehensive metrics for monitoring

### Metrics

The client package exports Prometheus metrics for monitoring:

#### Client Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `output_client_logs_total` | Counter | `endpoint` | Total logs sent by client |
| `dropped_logs_total` | Counter | `endpoint`, `reason` | Logs dropped (queue full, throttled, etc.) |
| `errors_total` | Counter | `type` | Errors by type |

#### DQue Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dque_queue_size` | Gauge | `endpoint` | Current queue size |
| `dque_batch_size` | Histogram | `endpoint` | Size of exported batches |
| `dque_export_duration_seconds` | Histogram | `endpoint`, `status` | Export operation duration |
| `dque_exports_total` | Counter | `endpoint`, `status` | Total exports by status |

#### gRPC Metrics (OTLP gRPC only)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `grpc_client_started_total` | Counter | `grpc_method`, `grpc_service` | RPCs started |
| `grpc_client_handled_total` | Counter | `grpc_method`, `grpc_service`, `grpc_code` | RPCs completed |
| `grpc_client_msg_sent_total` | Counter | `grpc_method`, `grpc_service` | Messages sent |
| `grpc_client_msg_received_total` | Counter | `grpc_method`, `grpc_service` | Messages received |

**Monitoring recommendations:**
- Alert on high `dropped_logs_total` rates
- Monitor `dque_queue_size` for queue buildup
- Track `dque_export_duration_seconds` for backend latency
- Watch `errors_total` for issues

## Examples

### Example 1: Basic OTLP gRPC Client

```go
package main

import (
    "context"
    "time"
    
    "github.com/gardener/logging/v1/pkg/client"
    "github.com/gardener/logging/v1/pkg/config"
    "github.com/gardener/logging/v1/pkg/types"
    "github.com/go-logr/logr"
)

func main() {
    cfg := config.Config{
        PluginConfig: config.PluginConfig{
            ShootType: "otlp_grpc",
        },
        OTLPConfig: config.OTLPConfig{
            Endpoint: "localhost:4317",
            Timeout:  30 * time.Second,
        },
    }
    
    ctx := context.Background()
    logger := logr.Discard()
    
    c, err := client.NewClient(ctx, cfg,
        client.WithTarget(client.Shoot),
        client.WithLogger(logger),
    )
    if err != nil {
        panic(err)
    }
    defer c.StopWait()
    
    // Send a log
    entry := types.OutputEntry{
        Timestamp: time.Now(),
        Record: map[string]any{
            "message": "Hello, World!",
            "level":   "info",
        },
    }
    
    if err := c.Handle(entry); err != nil {
        // Handle error
    }
}
```

### Example 2: OTLP HTTP with TLS

```go
cfg := config.Config{
    PluginConfig: config.PluginConfig{
        SeedType: "otlp_http",
    },
    OTLPConfig: config.OTLPConfig{
        Endpoint: "https://otlp-collector.example.com:4318",
        TLSCAFile: "/etc/ssl/certs/ca.pem",
        TLSCertFile: "/etc/ssl/certs/client.pem",
        TLSKeyFile: "/etc/ssl/private/client-key.pem",
        TLSMinVersion: "1.3",
    },
}

c, err := client.NewClient(ctx, cfg,
    client.WithTarget(client.Seed),
    client.WithLogger(logger),
)
```

### Example 3: Stdout Client for Debugging

```go
cfg := config.Config{
    PluginConfig: config.PluginConfig{
        ShootType: "STDOUT",
    },
}

c, err := client.NewClient(ctx, cfg,
    client.WithTarget(client.Shoot),
)
// Logs will be written to stdout in JSON format
```

### Example 4: High-Throughput Configuration

```go
cfg := config.Config{
    PluginConfig: config.PluginConfig{
        ShootType: "otlp_grpc",
    },
    OTLPConfig: config.OTLPConfig{
        Endpoint: "otlp-collector:4317",
        Compression: 1,  // Enable gzip
        
        // Larger batches for efficiency
        DQueBatchProcessorMaxQueueSize:   2048,
        DQueBatchProcessorMaxBatchSize:   512,
        DQueBatchProcessorExportInterval: 5 * time.Second,
        
        // DQue settings
        DQueConfig: config.DQueConfig{
            DQueDir:         "/var/log/fluent-bit-storage",
            DQueSegmentSize: 1000,
            DQueSync:        false,  // Async for performance
        },
        
        // Enable retry
        RetryEnabled:         true,
        RetryInitialInterval: 5 * time.Second,
        RetryMaxInterval:     30 * time.Second,
        RetryMaxElapsedTime:  5 * time.Minute,
    },
}
```

### Example 5: Rate-Limited Client

```go
cfg := config.Config{
    PluginConfig: config.PluginConfig{
        ShootType: "otlp_grpc",
    },
    OTLPConfig: config.OTLPConfig{
        Endpoint: "otlp-collector:4317",
        
        // Enable throttling
        ThrottleEnabled:        true,
        ThrottleRequestsPerSec: 100,  // Max 100 requests/sec
    },
}

c, err := client.NewClient(ctx, cfg,
    client.WithTarget(client.Shoot),
)

// Handle throttling
if err := c.Handle(entry); err != nil {
    if errors.Is(err, client.ErrThrottled) {
        // Log was throttled - consider buffering or dropping
    }
}
```

---

## Contributing

When adding new client types or modifying existing ones:

1. Implement the `OutputClient` interface
2. Add appropriate metrics
3. Write unit tests using Ginkgo and Gomega
4. Update this documentation
5. Follow coding standards and best practices

