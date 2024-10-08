# E2E Tests

E2E tests are written using [kubernetes-sigs/e2e-framework](https://github.com/kubernetes-sigs/e2e-framework).
The test suite builds the plugin container image and spins up a `kind` cluster with following components:
- Logging backends for shoot and seed logs
- Fluent-bit with the plugin under test
- Workloads producing logs in the seed and shoot control plane namespaces

Following test cases are covered by E2E tests:

## seed-logs

Verifies that log volumes produced by the workloads outside the shoot control plane namespaces
are correctly send to the backend storage in the seed namespace.

## shoot-logs

Verifies that log volumes produced by the workloads in the shoot control planes are correctly send to the backend storage in the same namespace.

## event-shoot-logs

Verifies that the kubernetes events from the workloads in the shoot control planes are correctly send to the backend storage in the same namespace. Verifies that the kubernetes events from the k8s-apiserver in the shoot control planes are correctly send to the backend storage in the same namespace.

## Dependencies 

The `e2e-framework` requires the following dependencies to be installed on the machine where the tests are executed:
- kind
- docker

## Running the tests
To run the tests, execute the following command:

```bash
go test ./tests/e2e/... --test.v
```
