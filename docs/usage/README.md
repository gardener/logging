# Logging Plugin

This guide is about Gardener Logging, how it is organized and how to use the dashboard to view the log data of Kubernetes clusters.

## Cluster level logging

Log data is fundamental for the successful operation activities of Kubernetes landscapes. It is used for investigating problems and monitoring cluster activity.

Cluster level logging is the recommended way to collect and store log data for Kubernetes cluster components. With cluster level logging the log data is externalized
in a logging backend where the log lifecycle management is independent from the lifecycle management of the Kubernetes resources.

Cluster level logging is not available by default with [Kubernetes](https://kubernetes.io/docs/concepts/cluster-administration/logging/#cluster-level-logging-architectures) and consumers have to additionally implement it.
The Kubernetes project only provides basic logging capabilities via `kubectl logs` where the kubelet keeps one terminated container with its logs.
When a pod is evicted from the node, all corresponding containers are also evicted, along with their logs.
This is why the default log storage solution is considered short-lived and not sufficient when you want to properly operate a Kubernetes environment.

Gardener, as an advanced Kubernetes management solution, follows the general recommendations and offers a cluster level logging solution to ensure proper log storage for all managed Kubernetes resources.
The log management is setup when a new cluster is created.
Log collection is organized using [fluent-bit](https://fluentbit.io).
Log storage and search is organized using [Vali](https://github.com/credativ/vali).
Log visualization is available using [Plutono](https://github.com/credativ/plutono) that is deployed with predefined dashboard and visualization for every shoot cluster.

Using Kubernetes operators can benefit from different capabilities like accessing the logs for
already terminated containers and performing fast and sophisticated search queries for investigating long-lasting or recurring problems based on logs from a long period of time.

In this guide, you will find out how to explore the log data for your clusters.

## Exploring logs

The sections below describe how access Grafana and use it to view the log data of your Kubernetes cluster.

### Accessing Plutono

Plutono UI is visible on the Shoot panel in the Gardner Dashboard App. Usually it follows a naming convention of the seeds clusters and can be bookmarked for convinience.

### Using Plutono

There are two options to explore log messages in Plutono.

#### Predefined Dashboards

The `Plutono` dashboards containing logs table are tagged with label `logging` for convinient dashboard filtering.

#### Explore tab

The second option is to use the **Explore** tab.

The explore tab allows filtering logs from the connected backend using the `Vali` LogQL. The latter is completly compatible with loki logql. The filters can be build either by selecting fields in the `Log Browser` or by entering the desired filters manuall. The UI supports auto completion of the filter names for convinience.

An Example filter are:

- `{pod_name="kube-apiserver-1234-1234"}` to select logs from the given pod
- `{pod_name=~"kube-apiserver.+"}` to use a regex in as pod name
- `sum(count_over_time({container_name="updater"}[5m]))` to aggregate logs count from a given container over time
