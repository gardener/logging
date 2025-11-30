# E2E Tests

E2E tests are based on [kubernetes-sigs/e2e-framework](https://github.com/kubernetes-sigs/e2e-framework).
The test suite builds the plugin container image and spins up a `kind` cluster with following components:

- Built fluent-bit container image with dockerfile target `fluent-bit-output`
- Victoria-Logs as Logging backends for shoot and seed logs
  - one instance for all shoot logs
  - one instance for seed logs
- otel-collector instances in for shoots and seed, otel-collector pushes logs to victoria-logs
 - one instance for all shoot logs
 - one instance for seed logs
- Fluent-bit with the plugin under test deployed as DaemonSet
- use fluent-bit.yaml configuration to configure the fluent-bit
- Fluent-bit plugins sent logs to the respective otel-collectors
- Simulate 100 shoots, where each shoot has:
  - a logger producing 1000 logs
  - a logging service pointing to the otel-collector for shoot
- A logger producing logs in the seed cluster
- A check counting the logs received in the victoria-logs instances

Following test cases are covered by E2E tests:

## seed-logs

Verifies that log volumes produced by the workloads outside the shoot control plane namespaces
are correctly send to the backend storage in the seed namespace.

## shoot-logs

Verifies that log volumes produced by the workloads in the shoot control planes are correctly send to the backend storage in the same namespace.

## Dependencies

The `e2e-framework` requires the following dependencies to be present on the machine where the tests are executed:

- kind
- docker

## Running the tests

To run all tests, execute the following command:

```bash
go test -v ./tests/e2e/...
```

Or to execute a given feature, run one of the following commands:

```bash
go test -v ./tests/e2e/... -args --feature "seed/logs"
go test -v ./tests/e2e/... -args --feature "shoot/logs"
```
