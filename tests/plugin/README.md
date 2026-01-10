# Plugin Integration Tests

This directory contains integration tests for the Gardener logging fluent-bit output plugin.

## Overview

This test simulates creating 100 clusters with a logger pod in each cluster,
verifying that the produced log volume is fully accounted.
The test creates an instance of a fluent-bit output plugin with a k8s informer
processing the cluster lifecycle events. Each cluster resource creates a dedicated output
client responsible for packing and sending the logs to the simulated backend.
Finally, the test counts the received logs in the backend and checks the total volume.

The test verifies the following plugin components:

- the plugin controller maintaining a list of clients corresponding to the cluster resources
- the seed and shoots client decorator chains
- the correct packaging of logs into respective backend streams

## Test Details

### Scenario

- **100 Gardener shoot clusters** are created as Kubernetes custom resources
- Each cluster generates **1,000 log entries** (100,000 total logs)
- The fluent-bit output plugin processes all logs with dynamic routing
- All logs are fully accounted through Prometheus metrics

### Architecture

- **Fake Kubernetes Client**: Uses `fakeclientset` with real informer factory
- **NoopClient**: Both seed and shoot clients use NoopClient for testing (no external dependencies)
- **Controller**: Watches cluster resources and creates clients dynamically
- **Worker Pool**: 10 parallel workers for efficient log generation
- **Metrics Validation**: Prometheus testutil ensures complete log accounting

### Performance

- **Runtime**: ~4-5 seconds for full test suite
- **Memory**: Minimal footprint with NoopClient
- **Disk I/O**: None (buffer disabled, direct mode)
- **Parallelism**: 10 worker goroutines

## Running the Tests

```bash
# Run all plugin tests
make test

# Run only the integration test
go test -v ./tests/plugin -run TestOutputPlugin -timeout=5m

# Run with verbose Ginkgo output
go test -v ./tests/plugin -ginkgo.v -timeout=5m
```

## Expected Results

✅ All 100 cluster resources created successfully  
✅ Controller creates 100 client instances  
✅ All 100,000 logs sent without error  
✅ IncomingLogs metric totals 100,000  
✅ DroppedLogs metric totals 100,000  
✅ Zero errors reported  

## Metrics Validated

- `fluentbit_gardener_incoming_logs_total{host="shoot--test--cluster-XXX"}`: Total should be 100,000
- `fluentbit_gardener_dropped_logs_total{host="http://logging.shoot--test--cluster-XXX.svc:4318/v1/logs"}`: Total should be 100,000
- `fluentbit_gardener_errors_total{type="*"}`: Should be 0

## Test Implementation

The test is implemented using:
- **Ginkgo v2**: BDD-style test framework
- **Gomega**: Matcher/assertion library
- **Ordered Container**: Tests run in sequence with dependencies
- **Worker Pool Pattern**: Efficient parallel log generation

See `plugin_test.go` for the full implementation.

