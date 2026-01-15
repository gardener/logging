# Troubleshooting Guide

This document provides solutions to common issues when using the Gardener Fluent Bit OTLP Output Plugin.

## Common Issues

### Logs Not Being Sent

#### Symptoms
- Logs are not appearing in the backend
- Queue size keeps growing
- No export errors in metrics

#### Troubleshooting Steps

1. **Check client type configuration**:
   ```ini
   SeedType OTLPGRPC  # or OTLPHTTP, stdout
   ShootType OTLPGRPC
   ```
   
   Verify the client type is set correctly. If empty, the plugin won't send logs.

2. **Verify endpoint connectivity**:
   ```bash
   # For gRPC
   grpcurl -plaintext victorialogs.logging.svc:4317 list
   
   # For HTTP
   curl -v https://victorialogs.logging.svc/insert/opentelemetry/v1/logs
   ```

3. **Check TLS configuration**:
   - Ensure certificate paths are correct
   - Verify certificate validity: `openssl x509 -in /path/to/cert.crt -text -noout`
   - Check server name matches certificate CN/SAN
   - Verify CA certificate is correct

4. **Review plugin logs**:
   ```bash
   # Increase log level in config
   LogLevel debug
   
   # Check Fluent Bit logs
   kubectl logs -n logging daemonset/fluent-bit
   ```

5. **Check backend availability**:
   ```bash
   # Test network connectivity
   nc -zv victorialogs.logging.svc 4317
   
   # Check DNS resolution
   nslookup victorialogs.logging.svc
   ```

### Queue Growing Continuously

#### Symptoms
- `dque_queue_size` metric keeps increasing
- Disk usage growing
- Logs being dropped (if queue full)

#### Root Causes and Solutions

1. **Backend performance issues**:
   - Backend may be too slow or unavailable
   - Check backend metrics and logs
   - Scale backend if needed

2. **Batch size too small**:
   ```ini
   # Increase batch size
   DQueBatchProcessorMaxBatchSize 512
   ```

3. **Export interval too high**:
   ```ini
   # Decrease export interval
   DQueBatchProcessorExportInterval 500ms
   ```

4. **Network latency**:
   - Check network connectivity to backend
   - Enable compression to reduce bandwidth:
   ```ini
   Compression 1
   ```

5. **Backend throttling**:
   - Check for rate limiting errors
   - Adjust throttle configuration:
   ```ini
   ThrottleEnabled true
   ThrottleRequestsPerSec 100
   ```

### High Memory Usage

#### Symptoms
- Pod OOMKilled
- High memory metrics
- System slowdown

#### Solutions

1. **Queue size too large**:
   ```ini
   # Reduce in-memory queue
   DQueBatchProcessorMaxQueueSize 256
   ```

2. **Batch size too large**:
   ```ini
   # Reduce batch size
   DQueBatchProcessorMaxBatchSize 128
   ```

3. **Too many clients**:
   - Check for client leaks with dynamic routing
   - Review client cleanup:
   ```ini
   DeletedClientTimeExpiration 30m
   ```

4. **Memory leak**:
   - Enable profiling and analyze heap:
   ```bash
   go tool pprof http://localhost:2021/debug/pprof/heap
   ```

### TLS/mTLS Errors

#### Symptoms
- "certificate verify failed" errors
- "tls: bad certificate" errors
- Connection refused

#### Solutions

1. **Certificate not found**:
   ```bash
   # Check file exists and is readable
   ls -la /etc/ssl/fluent-bit/tls.crt
   
   # Check permissions
   kubectl exec -it fluent-bit-xxx -- ls -la /etc/ssl/fluent-bit/
   ```

2. **Certificate expired**:
   ```bash
   # Check certificate validity
   openssl x509 -in /path/to/cert.crt -noout -dates
   ```

3. **CA certificate mismatch**:
   ```bash
   # Verify certificate chain
   openssl verify -CAfile /path/to/ca.crt /path/to/cert.crt
   ```

4. **Server name mismatch**:
   ```ini
   # Set correct server name for SNI
   TLSServerName victorialogs.logging.svc
   ```

5. **TLS version incompatibility**:
   ```ini
   # Adjust TLS version
   TLSMinVersion 1.2
   TLSMaxVersion 1.3
   ```

### Dynamic Routing Not Working

#### Symptoms
- Logs always go to default client
- Expected client not created
- "client not found" errors

#### Solutions

1. **Check JSONPath configuration**:
   ```ini
   # Verify path matches log structure
   DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
   ```

2. **Verify regex pattern**:
   ```ini
   # Test regex against namespace names
   DynamicHostRegex ^shoot--
   ```

3. **Check controller sync**:
   ```bash
   # Increase sync timeout
   ControllerSyncTimeout 120s
   ```

4. **Review cluster state routing**:
   ```ini
   # Ensure state-based routing is configured
   SendLogsToMainClusterWhenIsInReadyState true
   ```

5. **Check namespace metadata**:
   ```bash
   # Verify logs contain expected metadata
   LogLevel debug
   # Look for "extracted dynamic host" messages
   ```

### Export Errors

#### Symptoms
- `dque_export_errors_total` metric increasing
- "context deadline exceeded" errors
- "connection reset" errors

#### Solutions

1. **Timeout too short**:
   ```ini
   # Increase export timeout
   DQueBatchProcessorExportTimeout 60s
   Timeout 60s
   ```

2. **Backend overloaded**:
   - Reduce export rate:
   ```ini
   ThrottleEnabled true
   ThrottleRequestsPerSec 50
   ```

3. **Network issues**:
   - Check network connectivity
   - Test with smaller batches:
   ```ini
   DQueBatchProcessorMaxBatchSize 128
   ```

4. **Backend errors**:
   - Check backend logs for error details
   - Verify backend configuration

### Kubernetes Metadata Missing

#### Symptoms
- Logs missing pod_name, namespace, container_name
- Logs dropped when `DropLogEntryWithoutK8sMetadata` is true

#### Solutions

1. **Enable fallback to tag**:
   ```ini
   FallbackToTagWhenMetadataIsMissing true
   ```

2. **Check tag configuration**:
   ```ini
   TagKey tag
   TagPrefix kubernetes\\.var\\.log\\.containers
   TagExpression \\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$
   ```

3. **Verify Fluent Bit Kubernetes filter**:
   ```ini
   [Filter]
       Name kubernetes
       Match kubernetes.*
       Kube_URL https://kubernetes.default.svc:443
       Kube_CA_File /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
       Kube_Token_File /var/run/secrets/kubernetes.io/serviceaccount/token
   ```

4. **Check service account permissions**:
   ```bash
   # Verify RBAC allows reading pods/namespaces
   kubectl auth can-i get pods --as=system:serviceaccount:logging:fluent-bit
   ```

## Debug Mode

Enable comprehensive debugging:

```ini
[Output]
    Name gardener
    Match *
    LogLevel debug
    Pprof true
```

### Accessing Debug Information

1. **Metrics**:
   ```bash
   curl http://localhost:2021/metrics
   ```

2. **Health Check**:
   ```bash
   curl http://localhost:2021/healthz
   ```

3. **CPU Profile**:
   ```bash
   go tool pprof http://localhost:2021/debug/pprof/profile
   ```

4. **Heap Profile**:
   ```bash
   go tool pprof http://localhost:2021/debug/pprof/heap
   ```

5. **Goroutines**:
   ```bash
   curl http://localhost:2021/debug/pprof/goroutine?debug=2
   ```

## Performance Issues

### High CPU Usage

1. **Compression overhead**:
   ```ini
   # Disable compression if CPU-constrained
   Compression 0
   ```

2. **Too many regex operations**:
   - Simplify tag expressions
   - Use Kubernetes filter instead of tag parsing

3. **Too many clients**:
   - Review dynamic routing configuration
   - Reduce client count if possible

### High Disk I/O

1. **Frequent syncs**:
   ```ini
   # Disable fsync for better performance
   DQueSync false
   ```

2. **Small segments**:
   ```ini
   # Increase segment size
   DQueSegmentSize 1000
   ```

3. **Disk queue location**:
   - Use faster storage for queue directory
   - Consider tmpfs for ephemeral environments:
   ```ini
   DQueDir /dev/shm/fluent-bit-buffers
   ```

## Metrics Analysis

### Key Metrics to Monitor

```bash
# Queue health
curl -s http://localhost:2021/metrics | grep dque_queue_size

# Export performance
curl -s http://localhost:2021/metrics | grep dque_export_duration_seconds

# Error rate
curl -s http://localhost:2021/metrics | grep dque_export_errors_total

# Drop rate
curl -s http://localhost:2021/metrics | grep dque_dropped_total
```

### Alert Recommendations

1. **Queue Growing**:
   - Alert: `dque_queue_size > 400` for 5 minutes
   - Action: Investigate backend performance or increase export rate

2. **High Error Rate**:
   - Alert: `rate(dque_export_errors_total[5m]) > 0.1`
   - Action: Check backend connectivity and logs

3. **Logs Dropped**:
   - Alert: `rate(dque_dropped_total[5m]) > 0`
   - Action: Increase queue size or export rate

4. **High Export Latency**:
   - Alert: `histogram_quantile(0.95, dque_export_duration_seconds) > 10`
   - Action: Check network latency or backend performance

## Getting Help

If you've tried the solutions above and still have issues:

1. **Collect debug information**:
   ```bash
   # Plugin logs
   kubectl logs -n logging daemonset/fluent-bit --tail=1000 > fluent-bit.log
   
   # Metrics
   curl http://localhost:2021/metrics > metrics.txt
   
   # Configuration
   kubectl get configmap fluent-bit-config -o yaml > config.yaml
   ```

2. **Check known issues**:
   - GitHub Issues: [https://github.com/gardener/logging/issues](https://github.com/gardener/logging/issues)

3. **Ask for help**:
   - Create a GitHub issue with debug information
   - Join Gardener Slack: [#gardener](https://kubernetes.slack.com/messages/gardener)

## Additional Resources

- [Configuration Guide](configuration.md)
- [Architecture](architecture.md)
- [Usage Guide](usage.md)
- [pkg/client/README.md](../pkg/client/README.md)

