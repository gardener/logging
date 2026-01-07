# Configuration Guide

This document provides comprehensive configuration options for the Gardener Fluent Bit OTLP Output Plugin.

## Configuration Options

### OTLP Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `Endpoint` | OTLP endpoint URL (with or without scheme) | `localhost:4317` | string |
| `Insecure` | Use insecure connection (skip TLS) | `false` | bool |
| `Compression` | Compression algorithm (0=none, 1=gzip) | `0` | int |
| `Timeout` | Request timeout duration | `30s` | duration |
| `Headers` | Custom HTTP headers (format: `key1 value1,key2 value2`) | `{}` | map[string]string |

### Batch Processor Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `DQueBatchProcessorMaxQueueSize` | Maximum records in queue before dropping | `512` | int |
| `DQueBatchProcessorMaxBatchSize` | Maximum records per export batch | `256` | int |
| `DQueBatchProcessorExportTimeout` | Timeout for single export operation | `30s` | duration |
| `DQueBatchProcessorExportInterval` | Flush interval | `1s` | duration |
| `DQueBatchProcessorExportBufferSize` | Export buffer size | `10` | int |

### DQue (Disk Queue) Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `DQueDir` | Directory path for queue storage | `/tmp/flb-storage` | string |
| `DQueName` | Queue name (subdirectory under DQueDir) | `dque` | string |
| `DQueSegmentSize` | Number of entries per segment file | `500` | int |
| `DQueSync` | Sync writes to disk (true/false) | `false` | bool |

### Retry Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `RetryEnabled` | Enable retry logic | `true` | bool |
| `RetryInitialInterval` | Initial retry wait time | `5s` | duration |
| `RetryMaxInterval` | Maximum retry wait time | `30s` | duration |
| `RetryMaxElapsedTime` | Total time to retry before giving up | `1m` | duration |

### Throttle Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `ThrottleEnabled` | Enable rate limiting | `false` | bool |
| `ThrottleRequestsPerSec` | Maximum requests per second (0=unlimited) | `0` | int |

### TLS Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `TLSCertFile` | Path to client certificate | `""` | string |
| `TLSKeyFile` | Path to client private key | `""` | string |
| `TLSCAFile` | Path to CA certificate for server verification | `""` | string |
| `TLSServerName` | Server name for SNI | `""` | string |
| `TLSInsecureSkipVerify` | Skip certificate verification (insecure) | `false` | bool |
| `TLSMinVersion` | Minimum TLS version (1.0, 1.1, 1.2, 1.3) | `1.2` | string |
| `TLSMaxVersion` | Maximum TLS version | `""` (Go default) | string |

### Plugin Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `SeedType` | Client type for Seed clusters (`OTLPGRPC`/`OTLPHTTP`/`stdout`/`noop`) | `""` | string |
| `ShootType` | Client type for Shoot clusters (`OTLPGRPC`/`OTLPHTTP`/`stdout`/`noop`) | `""` | string |
| `LogLevel` | Plugin log level (debug, info, warn, error) | `info` | string |
| `Pprof` | Enable pprof profiling endpoints | `false` | bool |
| `HostnameValue` | Custom hostname to include in logs | OS hostname | string |
| `Origin` | Origin label for logs (seed/shoot identification) | `""` | string |

### Kubernetes Metadata Extraction

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `FallbackToTagWhenMetadataIsMissing` | Extract metadata from tag if missing | `false` | bool |
| `DropLogEntryWithoutK8sMetadata` | Drop logs without Kubernetes metadata | `false` | bool |
| `TagKey` | Record key containing the tag | `tag` | string |
| `TagPrefix` | Tag prefix (metadata not searched here) | `kubernetes\\.var\\.log\\.containers` | string |
| `TagExpression` | Regex to extract pod/namespace/container from tag | `\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$` | string |

### Dynamic Routing Configuration

| Key | Description | Default | Type |
|-----|-------------|---------|------|
| `DynamicHostPath` | JSONPath to extract dynamic host from log metadata | `""` | string |
| `DynamicHostPrefix` | Prefix for dynamic host URL | `""` | string |
| `DynamicHostSuffix` | Suffix for dynamic host URL | `""` | string |
| `DynamicHostRegex` | Regex to validate dynamic host | `*` | string |
| `ControllerSyncTimeout` | Time to wait for cluster object sync | `60s` | duration |
| `DeletedClientTimeExpiration` | Expiration time for deleted cluster clients | `1h` | duration |

### Cluster State-Based Routing

Control where logs are sent based on Shoot cluster state:

| Cluster State | Send to Seed | Send to Shoot (Dynamic) |
|---------------|--------------|-------------------------|
| Creation | `SendLogsToDefaultClientWhenClusterIsInCreationState` (true) | `SendLogsToMainClusterWhenIsInCreationState` (true) |
| Ready | `SendLogsToDefaultClientWhenClusterIsInReadyState` (false) | `SendLogsToMainClusterWhenIsInReadyState` (true) |
| Hibernating | `SendLogsToDefaultClientWhenClusterIsInHibernatingState` (false) | `SendLogsToMainClusterWhenIsInHibernatingState` (false) |
| Hibernated | `SendLogsToDefaultClientWhenClusterIsInHibernatedState` (false) | `SendLogsToMainClusterWhenIsInHibernatedState` (false) |
| Deletion | `SendLogsToDefaultClientWhenClusterIsInDeletionState` (true) | `SendLogsToMainClusterWhenIsInDeletionState` (true) |
| Restore | `SendLogsToDefaultClientWhenClusterIsInRestoreState` (true) | `SendLogsToMainClusterWhenIsInRestoreState` (true) |
| Migration | `SendLogsToDefaultClientWhenClusterIsInMigrationState` (true) | `SendLogsToMainClusterWhenIsInMigrationState` (true) |

## Configuration Examples

### Basic OTLP gRPC Configuration

Send logs to a VictoriaLogs backend using OTLP over gRPC:

```ini
[Output]
    Name gardener
    Match kubernetes.*
    
    # Client type selection
    SeedType OTLPGRPC
    ShootType OTLPGRPC
    
    # OTLP endpoint
    Endpoint victorialogs.logging.svc.cluster.local:4317
    Insecure false
    Compression 1
    Timeout 30s
    
    # Batch processing
    DQueBatchProcessorMaxQueueSize 512
    DQueBatchProcessorMaxBatchSize 256
    DQueBatchProcessorExportTimeout 30s
    DQueBatchProcessorExportInterval 1s
    
    # Disk queue configuration
    DQueDir /fluent-bit/buffers/otlp
    DQueName gardener-otlp
    DQueSegmentSize 500
    DQueSync false
    
    # Retry configuration
    RetryEnabled true
    RetryInitialInterval 5s
    RetryMaxInterval 30s
    RetryMaxElapsedTime 1m
    
    # TLS configuration
    TLSCertFile /etc/ssl/certs/client.crt
    TLSKeyFile /etc/ssl/private/client.key
    TLSCAFile /etc/ssl/certs/ca.crt
    TLSMinVersion 1.2
    
    # Plugin settings
    LogLevel info
    HostnameValue seed-cluster-1
    Origin seed
```

### OTLP HTTP Configuration with Custom Headers

Send logs using OTLP over HTTP with authentication headers:

```ini
[Output]
    Name gardener
    Match kubernetes.*
    
    # Use HTTP client
    SeedType OTLPHTTP
    
    # OTLP HTTP endpoint
    Endpoint https://victorialogs.example.com/insert/opentelemetry/v1/logs
    Insecure false
    Timeout 30s
    
    # Custom headers for authentication
    Headers Authorization Bearer YOUR_TOKEN,X-Scope-OrgID tenant-1
    
    # Batch processing
    DQueBatchProcessorMaxBatchSize 256
    DQueBatchProcessorExportInterval 5s
    
    # Disk queue
    DQueDir /fluent-bit/buffers
    DQueName otlp-http
    
    # Rate limiting
    ThrottleEnabled true
    ThrottleRequestsPerSec 100
```

### Dynamic Multi-Cluster Routing

Route logs to different backends based on Shoot cluster namespaces:

```ini
[Output]
    Name gardener
    Match kubernetes.*
    
    # Default Seed cluster client
    SeedType OTLPGRPC
    Endpoint victorialogs-seed.logging.svc:4317
    
    # Dynamic Shoot cluster routing
    ShootType OTLPGRPC
    DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
    DynamicHostPrefix victorialogs.
    DynamicHostSuffix .svc.cluster.local:4317
    DynamicHostRegex ^shoot--
    
    # Cluster state-based routing
    SendLogsToMainClusterWhenIsInReadyState true
    SendLogsToMainClusterWhenIsInHibernatingState false
    SendLogsToMainClusterWhenIsInHibernatedState false
    SendLogsToDefaultClientWhenClusterIsInReadyState false
    SendLogsToDefaultClientWhenClusterIsInHibernatingState true
    
    # Kubernetes metadata extraction
    FallbackToTagWhenMetadataIsMissing true
    DropLogEntryWithoutK8sMetadata true
    TagKey tag
    TagPrefix kubernetes\\.var\\.log\\.containers
    TagExpression \\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$
    
    # Buffer configuration
    DQueDir /fluent-bit/buffers/shoot
    DQueName gardener-shoot
    DQueSegmentSize 500
    
    # Controller settings
    ControllerSyncTimeout 60s
    DeletedClientTimeExpiration 1h
```

### Development/Debug Configuration

Use stdout client for local development:

```ini
[Output]
    Name gardener
    Match *
    
    # Use stdout for debugging
    SeedType stdout
    ShootType stdout
    
    LogLevel debug
    Pprof true
```

### Production Configuration with mTLS

Full production setup with mutual TLS authentication:

```ini
[Output]
    Name gardener
    Match kubernetes.*
    
    # Production settings
    SeedType OTLPGRPC
    Endpoint victorialogs-prod.logging.svc:4317
    LogLevel info
    
    # mTLS configuration
    Insecure false
    TLSCertFile /etc/ssl/fluent-bit/tls.crt
    TLSKeyFile /etc/ssl/fluent-bit/tls.key
    TLSCAFile /etc/ssl/fluent-bit/ca.crt
    TLSServerName victorialogs-prod.logging.svc
    TLSInsecureSkipVerify false
    TLSMinVersion 1.3
    
    # Optimized batch processing
    DQueBatchProcessorMaxQueueSize 1024
    DQueBatchProcessorMaxBatchSize 512
    DQueBatchProcessorExportTimeout 30s
    DQueBatchProcessorExportInterval 2s
    DQueBatchProcessorExportBufferSize 10
    
    # Persistent buffering
    DQueDir /var/fluent-bit/buffers
    DQueName gardener-prod
    DQueSegmentSize 1000
    DQueSync true
    
    # Retry with exponential backoff
    RetryEnabled true
    RetryInitialInterval 10s
    RetryMaxInterval 5m
    RetryMaxElapsedTime 10m
    
    # Rate limiting
    ThrottleEnabled true
    ThrottleRequestsPerSec 500
    
    # Compression
    Compression 1
    
    # Monitoring
    HostnameValue ${HOSTNAME}
    Origin seed
```

