# Architecture

This document describes the architecture of the Gardener Fluent Bit OTLP Output Plugin.

![Gardener OTLP Plugin Architecture](images/gardener-logging-otlp-plugin.png)

## OTLP Clients

The plugin supports multiple client implementations:

- **OTLP gRPC Client**: High-performance binary protocol with bi-directional streaming
- **OTLP HTTP Client**: HTTP-based transport for environments where gRPC is not available
- **Stdout Client**: JSON output to stdout for debugging
- **Noop Client**: No-operation client for testing

### OTLP gRPC Client

High-performance OTLP over gRPC transport:
- Binary protocol (Protobuf)
- Bi-directional streaming
- Efficient compression
- Recommended for production

**Use cases:**
- Production environments
- High-volume log shipping
- Low-latency requirements
- When backend supports gRPC

### OTLP HTTP Client

HTTP-based OTLP transport:
- Works through HTTP proxies
- Firewall-friendly
- HTTP/1.1 and HTTP/2 support
- JSON or Protobuf encoding

**Use cases:**
- When gRPC is not available or blocked by firewalls
- HTTP proxy environments
- Debugging (easier to inspect with standard tools)
- When backend only supports HTTP

### Stdout Client

Debug client that writes logs to stdout:
- JSON formatted output
- Useful for local development
- No external dependencies

### Noop Client

No-operation client:
- Discards all logs
- Useful for testing and benchmarking
- Zero overhead

## DQue Batch Processor

The plugin uses a disk-based queue (dque) for persistent buffering:

- **Persistent Storage**: Logs are stored on disk to survive process restarts
- **Batch Processing**: Logs are batched for efficient export
- **Backpressure Handling**: Queue prevents memory exhaustion under high load
- **Configurable Sync**: Optional fsync for data durability

### How It Works

1. **Enqueue**: Incoming logs are added to the disk-backed queue
2. **Batch Formation**: Logs are accumulated until batch size or timeout is reached
3. **Export**: Batches are exported to the OTLP backend
4. **Retry**: Failed exports are retried with exponential backoff
5. **Dequeue**: Successfully exported logs are removed from the queue

### Performance Tuning

- **Queue Size**: Controls maximum in-memory records before dropping
- **Batch Size**: Larger batches improve throughput but increase latency
- **Export Interval**: Shorter intervals reduce latency but increase overhead
- **Segment Size**: Affects disk I/O patterns and memory usage

## Dynamic Routing Controller

The controller watches Kubernetes cluster resources and manages client routing:

- **Cluster State Monitoring**: Tracks Shoot cluster lifecycle states
- **Dynamic Client Creation**: Creates clients for new namespaces matching patterns
- **Client Cleanup**: Removes clients for deleted clusters after expiration
- **State-Based Routing**: Routes logs based on cluster state (Ready, Hibernating, etc.)

### Routing Logic

1. **Metadata Extraction**: Extract cluster information from log metadata (namespace, labels)
2. **Client Selection**: Determine appropriate client based on:
   - Cluster namespace pattern (e.g., `shoot--*`)
   - Cluster state (Ready, Hibernating, Hibernated, etc.)
   - Configuration rules
3. **Dynamic Host Resolution**: Build endpoint URL from prefix, extracted value, and suffix
4. **Client Management**: Create, cache, and cleanup clients as needed

### Cluster State Handling

Different cluster states have different routing behaviors:

- **Creation**: Logs can be sent to both Seed and Shoot (configurable)
- **Ready**: Typically only to Shoot cluster
- **Hibernating/Hibernated**: Typically only to Seed cluster (Shoot unavailable)
- **Deletion**: Logs sent to both for troubleshooting
- **Restore/Migration**: Logs sent to both for observability during transitions

## Component Interaction

```
┌─────────────────┐
│   Fluent Bit    │
│   (Input)       │
└────────┬────────┘
         │
         │ Log Records
         ▼
┌─────────────────────────────────────────────────────┐
│            Gardener Output Plugin                   │
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Record Converter                            │ │
│  │  - Extract Kubernetes metadata               │ │
│  │  - Convert to OTLP log records              │ │
│  └──────────────┬───────────────────────────────┘ │
│                 │                                   │
│                 ▼                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │  Dynamic Routing Controller                  │ │
│  │  - Determine target client                   │ │
│  │  - Create clients on-demand                  │ │
│  └──────────────┬───────────────────────────────┘ │
│                 │                                   │
│                 ▼                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │  Client (OTLP gRPC/HTTP/stdout/noop)        │ │
│  │  - DQue batch processor                      │ │
│  │  - Persistent buffering                      │ │
│  │  - Retry logic                               │ │
│  │  - Rate limiting                             │ │
│  └──────────────┬───────────────────────────────┘ │
└─────────────────┼───────────────────────────────────┘
                  │
                  │ OTLP Protocol
                  ▼
         ┌────────────────┐
         │  Backend       │
         │  (VictoriaLogs,│
         │   Loki, etc.)  │
         └────────────────┘
```

## Metrics & Observability

The plugin exposes Prometheus metrics on port 2021:

- **Queue Metrics**: `dque_queue_size`, `dque_enqueued_total`, `dque_dequeued_total`
- **Export Metrics**: `dque_export_duration_seconds`, `dque_export_errors_total`
- **Client Metrics**: OTLP SDK metrics for export operations
- **Health Checks**: `/healthz` endpoint for probes
- **Profiling**: Optional pprof endpoints on `/debug/pprof/`

### Metrics Endpoints

- `GET /metrics` - Prometheus metrics
- `GET /healthz` - Health check (returns 200 OK when healthy)
- `GET /debug/pprof/` - Profiling endpoints (when enabled)

## Security

### TLS/mTLS

The plugin supports full TLS configuration:

- **Client Certificates**: mTLS authentication with client certificates
- **CA Verification**: Server certificate validation with custom CA
- **SNI**: Server Name Indication support
- **Version Control**: Configurable minimum and maximum TLS versions

### Authentication

- **Header-based**: Custom headers for bearer tokens or API keys
- **Certificate-based**: mTLS with client certificates
- **No authentication**: For testing or internal networks

## Performance Considerations

### Memory Usage

- Queue size controls in-memory buffer
- Batch size affects memory per export operation
- Multiple clients multiply memory requirements

### Disk Usage

- Queue segments stored on disk
- Segment size affects file count and I/O patterns
- Cleanup happens after successful export

### Network

- Compression reduces bandwidth (at CPU cost)
- Batch size affects request frequency
- Rate limiting prevents backend overload

### CPU

- Compression increases CPU usage
- Batch processing reduces overhead
- Multiple clients increase context switching

