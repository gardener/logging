# E2E Tests

E2E tests are based on [kubernetes-sigs/e2e-framework](https://github.com/kubernetes-sigs/e2e-framework).
The test suite builds the plugin container image and spins up a `kind` cluster with following components:

- Built fluent-bit-plugin container image with dockerfile target `fluent-bit-plugin`
- Built event-logger container image with dockerfile target `event-logger`
- Victoria-Logs as the logging backend (single instance for all logs)
- Fluent-bit with the plugin under test deployed as DaemonSet
- Fluent-bit configuration (`fluent-bit.yaml`) with:
  - Tail input for container logs
  - Systemd input for kubelet and containerd logs
  - Gardener output plugin sending logs directly to Victoria-Logs via OTLP HTTP
- Simulate 100 shoots, where each shoot has:
  - A namespace `shoot--logging--dev-XX`
  - A logging service (ExternalName) pointing to Victoria-Logs
  - A Cluster custom resource
  - A log-generator job producing 1000 logs
- A log-generator job producing 1000 logs in the seed (fluent-bit) namespace
- A curl-based fetcher deployment for querying Victoria-Logs

Following test cases are covered by E2E tests:

## shoot/logs

Verifies that log volumes produced by the workloads in the shoot control planes are correctly sent to the backend storage.
- Creates 100 shoot namespaces with log-generator jobs (1000 logs each)
- Validates per-namespace log counts (>=1000 logs per namespace)
- Validates total shoot log count (>=100,000 logs)

## seed/logs

Verifies that log volumes produced by the workloads in the seed cluster (outside shoot control plane namespaces)
are correctly sent to the backend storage.
- Creates a log-generator job in the fluent-bit namespace (1000 logs)
- Validates seed log count (>=1000 logs)

## systemd/logs

Verifies that systemd journal logs from kubelet and containerd services are correctly captured and sent to the backend storage.
- Queries Victoria-Logs for kubelet.service logs
- Queries Victoria-Logs for containerd.service logs
- Validates both services have log entries

## event-logger/basic

Verifies that the event-logger component correctly captures Kubernetes events and forwards them to Victoria-Logs.
- Deploys event-logger in the fluent-bit namespace with proper RBAC
- Creates a test Kubernetes event in the fluent-bit namespace
- Validates the event appears in Victoria-Logs

## Dependencies

The `e2e-framework` requires the following dependencies to be present on the machine where the tests are executed:

- kind
- docker

## Running the tests

To run all tests, execute the following command:

```bash
go test -v ./tests/e2e/...
```

Or to execute a specific feature, run one of the following commands:

```bash
go test -v ./tests/e2e/... -args --feature "seed/logs"
go test -v ./tests/e2e/... -args --feature "shoot/logs"
go test -v ./tests/e2e/... -args --feature "systemd/logs"
go test -v ./tests/e2e/... -args --feature "event-logger/basic"
```
