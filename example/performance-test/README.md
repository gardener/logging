# Logging Ingestion Performance Test Harness

This directory contains a lightweight harness for stress / performance validation of the Fluent Bit → Vali (Loki‑compatible) logging pipeline by generating synthetic log volume across multiple simulated shoot clusters.

---

## 1. Objectives

* Validate end‑to‑end ingestion throughput.
* Observe propagation delay (time until logs are queryable in Vali).
* Exercise multi‑cluster (namespace‑scoped) fan‑out of log sources.
* Provide reproducible parameters to scale horizontal concurrency and per‑pod log volume.

---

## 2. High‑Level Flow

1. For each index `i = 1..CLUSTERS`:

* Namespace `shoot--logging--dev-<i>` created.
* ExternalName Service `logging` points to central Vali endpoint `logging-vali-shoot.fluent-bit.svc.cluster.local`.
* A cluster CR (based on `cluster/cluster.yaml`) is patched (names) and applied.
* A Kubernetes Job `logger` with `parallelism = completions = JOBS` starts `JOBS` pods emitting synthetic logs.

* Each logger pod runs `nickytd/log-generator:latest` with `--wait` and `--count` flags.
* Fluent Bit DaemonSet (deployed separately by the chart) tails container logs and forwards to Vali.
* Validation (`fetch.sh`) queries Vali per cluster/namespace using:

```promql
sum(count_over_time({namespace_name=\"shoot--logging--dev-${i}\"}[24h]))
```

The script polls aggregated logs count per namespace. (CLUSTERS)
Expected count count ≈ `JOBS * LOGS` (after all pods finish and ingestion catches up).

---

## 3. Key Components

| Component | Purpose | Key Identifiers |
|-----------|---------|-----------------|
| Namespaces | Isolate simulated shoot clusters | `shoot--logging--dev-<i>` |
| ExternalName Service | Local DNS handle to central Vali | Service name `logging` → `logging-vali-shoot.fluent-bit.svc.cluster.local` |
| Cluster CR (patched) | Simulates Gardener shoot metadata | Patched via `yq` in `up.sh` |
| Logger Job | Generates logs (one per namespace) | Job `logger` with `parallelism` & `completions` = `JOBS` |
| Logger Pods | Emit `LOGS` lines each after `LOGS_DELAY` | Container name `logger` (+ automatic pod hash) |
| Fluent Bit DS | Collects & ships logs | Configured elsewhere in chart |
| Vali | Loki‑compatible storage/query | Queried via `logcli` |
| check.sh | Validation / metric polling | Uses PromQL‑style log query |

## 4. Environment Parameters

Set as shell environment variables before running scripts:

| Variable | Description | Effect on Volume / Concurrency | Typical Adjustment |
|----------|-------------|--------------------------------|--------------------|
| `CLUSTERS` | Number of simulated clusters (namespaces) | Linear multiplier of total logs | Increase to test horizontal scalability |
| `JOBS` | Pods per logger Job (`parallelism` & `completions`) | Multiplier per cluster | Increase to simulate intra‑cluster fan‑out |
| `LOGS` | Lines emitted per pod | Direct volume per pod | Increase for higher per‑pod throughput test |
| `LOGS_DELAY` | Seconds pods wait before emitting | Staggers start; warms pipeline | Increase if startup storms cause backpressure |
| `QUERY_WAIT` | Initial delay before first query (default 1) | Affects earliest validation time | Increase if ingestion latency > 1s |
| `QUERY_RETRIES` | Max polling attempts (default 10) | Longer patience for data arrival | Increase for slow or large clusters |
| `QUERY_INTERVAL` | Seconds between retries (default 5) | Total wait window = `QUERY_WAIT + QUERY_RETRIES * QUERY_INTERVAL` | Tune for responsiveness vs API load |

Derived expectation:

```text
EXPECTED_TOTAL = CLUSTERS * JOBS * LOGS
```

## 5. Scripts

### up.sh

Creates namespaces, services, cluster CRs, and logger Jobs in parallel for speed.

Order (each batched with background `&` + `wait` barrier):

1. Namespaces
2. ExternalName Services
3. Cluster CRs
4. Logger Jobs

### check.sh

Polls Vali using `logcli` and prints last cumulative count sample with timestamp. (Adjust date command for macOS: replace `date -d` with `date -r`.)

### fetch.sh

Fetches aggregated logs count per namespace from Vali.

## 6. Observations & Troubleshooting

Observe Fluent-Bit logs for issues on input and output plugins:

```text
sending batch, will retry" status=429 error="server returned HTTP status 429 Too Many ││ Requests (429): Ingestion rate limit exceeded (limit: 4194304 bytes/sec) while attempting to ingest '221' lines totaling '30 ││ 719' bytes, reduce log volume or contact your Loki administrator to see if the limit can be increased"
```

Usually signals for pressure on the receiving end (Vali). Consider reducing `LOGS` or `JOBS`, or increasing Vali resources/limits.

```text
fluent-bit [2025/05/25 05:25:05.196916584] [ info] [input] tail.2 resume (mem buf overlimit - buf size 29693936B now below limit 30000000B) │ │ fluent-bit [2025/05/25 05:25:05.196916584] [ warn] [input] tail.2 paused (mem buf overlimit - event of size 327B exceeded limit 30000000 to 30000119B) │ │ fluent-bit [2025/10/08 08:45:12.330230461] [ info] [input] pausing tail.2
```

Usual signals for input backpressure due to high memory buffer usage. Consider reducing `LOGS` or `JOBS`, or increasing Fluent Bit memory limits in tail plugin config.

Portforward plutono port and observe fluent-bit dashboard.
`[Fluentbit] Output Records Per Second` reveals actual thrioughput:

using following prometheus query:

```promql
sum(rate(fluentbit_output_proc_records_total{pod=~\"$pod\"}[$__rate_interval])) by (pod, name)
```

During run after gradual rampup the throughput should stabilize around a certain value.

## 7. Performance K8S Cluster sizing

For high load tests, ensure the Kubernetes cluster hosting Fluent Bit and Vali is appropriately sized. At least 6 nodes with 4CPU Cores and 16GB RAM each are recommended to handle the load without resource contention. Since all logs are sent to a single Vali instance, ensure it has sufficient resources to manage the ingestion rate.

## 8. Cleanup

To cleanup environment, run:

```bash
./down.sh
helm delete logging -n fluent-bit
kubectl delete pvc --all -n fluent-bit
```

## 8. Quick Reference

```text
TOTAL LOGS = CLUSTERS * JOBS * LOGS

vali_query: sum(count_over_time({namespace_name="shoot--logging--dev-X"}[24h]))
vali_config: [vali-config.yaml](charts/fluent-bit-plugin/templates/vali-config.yaml)

fluent-bit_config: [fluent-bit-config.yaml](charts/fluent-bit-plugin/templates/fluent-bit-config.yaml)
```
