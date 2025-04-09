# GEP-[n]: Introducing OpenTelemetry Operator and Collectors in Shoot Control Planes

**title:** Introducing OpenTelemetry Operator and Collectors in Shoot Control Planes
**gep-number:** [n]
**creation-date:** 20XX-XX-XX
**status:** implementable
**authors:**
- "@nickytd"
- ""
**reviewers:**
- ""

---

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Alternatives](#alternatives)

---

## Summary

This proposal introduces the OpenTelemetry operator and deploys OpenTelemetry collectors within the shoot control plane namespaces of Gardener-managed clusters. Building on the foundation laid by *GEP-19* (migration to Prometheus and Fluent-bit operators) and the *Observability 2.0* vision, this GEP advances Gardener's observability stack by adopting OpenTelemetry standards. The initiative aims to layout the foundation for building a more cohesive and interoperable observability framework.

---

## Motivation

Gardener's current observability stack, while improved by the operator-based approach in *GEP-19*, still relies on vendor specific format and protocols for collecting, processing and storing observability signals. That brings challenges in extending both components and scenarios use to process these signals and enforces lock-in integration scenarios with external consumers. By introducing the OpenTelemetry operator and collectors, this proposal addresses these challenges, aligning Gardener with industry trends and preparing it for future observability enhancements, such as unified visualization and tracing support.

### Goals

- **Deploy the OpenTelemetry Operator on Seed Clusters:** Install and configure the OpenTelemetry operator to manage otel-collector instances across shoot control plane namespaces.
- **Deploy Gardener OpenTelemetry Collector Distribution in Shoot Control Planes:** Provision a dedicated Gardener OpenTelemetry collector distribution instance in each shoot control plane namespace to handle observability data.
- **Transport Logs signals through the OpenTelemetry Collector Distribution:** Configure fluent-bits to send logs to the OTel Collectors receivers and wire the otel collector exporters to current logging backend.
- **Replace valitail system service with otel-collectors on shoot nodes:** -


### Non-Goals

- **Immediate Replacement of Existing Tools:** This proposal does not aim to immediately decommission tools like Vali, Fluent-bit, or Prometheus; the migration will be phased.
- **Unified Visualization Implementation:** Developing a single UI for logs, metrics, and traces is out of scope, as no mature open-source solution currently exists (per *Observability 2.0*).
- **Full Tracing Adoption:** While tracing support is enabled, widespread instrumentation of Gardener components for traces is a future step.

---

## Proposal

This section outlines the implementation plan to introduce the OpenTelemetry operator and collectors into Gardener's shoot control plane namespaces, aligning with Step 2 of the *Observability 2.0* roadmap.

### 1. Deploy the OpenTelemetry Operator on Seed Clusters

- **Objective:** Establish centralized management of OpenTelemetry collectors across all shoot control planes.
- **Details:** Deploy the OpenTelemetry operator (e.g., from the [OpenTelemetry Operator for Kubernetes](https://github.com/open-telemetry/opentelemetry-operator)) in a dedicated namespace (e.g., `monitoring`) on each seed cluster using `ManagedResources`, similar to the Prometheus operator deployment in *GEP-19*.
- **Configuration:** The operator will use Kubernetes custom resources (CRs) to define and manage collector instances, ensuring scalability and consistency.

### 2. Create OpenTelemetry Collector Instances per Shoot

- **Objective:** Provide each shoot control plane with a dedicated collector to process its observability data.
- **Details:** For each shoot, deploy an OpenTelemetry collector instance in its control plane namespace (e.g., `shoot--project--name`). The collector will run as a `Deployment` managed by the OpenTelemetry operator.
- **Example CR:**
  ```yaml
  apiVersion: opentelemetry.io/v1beta1
  kind: OpenTelemetryCollector
  metadata:
    name: shoot-otel-collector
    namespace: shoot--project--name
  spec:
    mode: deployment
    serviceAccount: shoot-otel-collector
    config:
      receivers:
        # Configuration defined in step 3
      processors:
        batch:
          timeout: 5s
          send_batch_size: 1000
      exporters:
        # Configuration defined in step 3
      service:
        pipelines:
          logs:
            receivers: [loki, k8s_events]
            processors: [batch]
            exporters: [otlphttp]
          metrics:
            receivers: [prometheus]
            processors: [batch]
            exporters: [otlphttp]
  ```

### 3. Configure the Collectors

- **Objective:** Enable collectors to handle logs, metrics, and traces from shoot components.
- **Details:** Configure each collector with:
  - **Receivers:**
    - `loki`: Receive logs from Fluent-bit (via Vali-compatible endpoint).
    - `k8s_events`: Capture Kubernetes events from the shoot control plane.
    - `prometheus`: Scrape metrics from Prometheus endpoints.
  - **Processors:** Use `batch` to optimize data processing and reduce export frequency.
  - **Exporters:** Export data in OTLP format (e.g., via `otlphttp`) to a backend like VictoriaLogs or an interim aggregator.
- **Sample Configuration (embedded in CR):**
  ```yaml
  config:
    receivers:
      loki:
        protocols:
          http:
            endpoint: "0.0.0.0:3100"
      k8s_events:
        namespaces:
          - "shoot--project--name"
      prometheus:
        config:
          scrape_configs:
            - job_name: "shoot-components"
              kubernetes_sd_configs:
                - role: pod
                  namespaces:
                    names:
                      - "shoot--project--name"
    exporters:
      otlphttp:
        logs_endpoint: "http://victorialogs:9428/insert/opentelemetry/v1/logs"
        metrics_endpoint: "http://victoriametrics:8428/api/v1/write"
    service:
      pipelines:
        logs:
          receivers: [loki, k8s_events]
          processors: [batch]
          exporters: [otlphttp]
        metrics:
          receivers: [prometheus]
          processors: [batch]
          exporters: [otlphttp]
  ```

### 4. Integrate with Existing Components

- **Objective:** Ensure a smooth transition by interfacing with the current observability stack.
- **Details:**
  - **Fluent-bit Integration:** Configure Fluent-bit to forward logs to the collector’s `loki` receiver, maintaining compatibility with the Vali-based setup from *GEP-19*.
  - **Prometheus Integration:** Use the `prometheus` receiver to scrape metrics from existing Prometheus targets, preserving *GEP-19*’s monitoring configurations.
  - **Event Logger Replacement:** The `k8s_events` receiver obsoletes the separate event logger by capturing events directly.
- **Migration Path:** Initially, collectors can act as intermediaries, forwarding data to existing backends (e.g., Vali, Prometheus), with plans to transition to OTLP-native backends like VictoriaLogs in future steps.

### 5. Define Custom Resources for Flexibility

- **Objective:** Allow tailored configurations for each shoot’s observability needs.
- **Details:** Utilize the OpenTelemetry operator’s CRDs to define collector configurations declaratively. Extensions can supply additional CRs (e.g., `OpenTelemetryCollector` overrides) to customize pipelines, similar to *GEP-19*’s BYOMC/BYOLC approach.
- **Example Extension CR:**
  ```yaml
  apiVersion: opentelemetry.io/v1beta1
  kind: OpenTelemetryCollector
  metadata:
    name: shoot-otel-collector-extension
    namespace: shoot--project--name
  spec:
    config:
      receivers:
        filelog:
          include: ["/var/log/extension/*.log"]
      service:
        pipelines:
          logs/extension:
            receivers: [filelog]
            exporters: [otlphttp]
  ```

---

## Alternatives

### Direct Collector Deployment Without an Operator

- **Description:** Deploy OpenTelemetry collectors manually in each shoot control plane namespace without using an operator.
- **Pros:** Simplifies initial setup, avoids operator overhead.
- **Cons:** Lacks scalability and management features (e.g., automatic upgrades, CR-based configuration), which are critical in Gardener’s multi-tenant environment.
- **Reason Rejected:** The operator aligns with *GEP-19*’s precedent of using operators for observability components, ensuring consistency and scalability.

### Delaying OpenTelemetry Adoption

- **Description:** Postpone OpenTelemetry integration until a unified visualization tool matures (e.g., CNCF Perses).
- **Pros:** Avoids interim solutions, potentially integrates collection and visualization simultaneously.
- **Cons:** Delays benefits of standardized data collection and correlation, slowing Gardener’s progress toward *Observability 2.0*.
- **Reason Rejected:** Early adoption of OpenTelemetry for data collection is a foundational step, independent of visualization maturity, and aligns with the phased roadmap.

---

This GEP establishes a critical milestone in Gardener’s journey toward *Observability 2.0*. By deploying the OpenTelemetry operator and collectors, Gardener gains a standardized, extensible observability framework that integrates with its existing stack while paving the way for future enhancements like tracing and OTLP-native backends.