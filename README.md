# Fluent Bit output plugin

![Logging Logo](images/logo/logging.png)

[![REUSE status](https://api.reuse.software/badge/github.com/gardener/logging)](https://api.reuse.software/info/github.com/gardener/logging)
[![CI Build status](https://concourse.ci.gardener.cloud/api/v1/teams/gardener/pipelines/logging-master/jobs/master-head-update-job/badge)](https://concourse.ci.gardener.cloud/teams/gardener/pipelines/logging-master/jobs/master-head-update-job)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/logging)](https://goreportcard.com/report/github.com/gardener/logging)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE) [![Release](https://img.shields.io/github/v/release/gardener/logging.svg?style=flat)](https://github.com/gardener/logging) [![Go Reference](https://pkg.go.dev/badge/github.com/gardener/logging.svg)](https://pkg.go.dev/github.com/gardener/logging)

This plugin extends [Fluent Bit output plugin](https://github.com/credativ/vali/tree/main/cmd/fluent-bit) which aims to forward log messages from fluent-bit to Vali backend.
Тhe plugin meets the needs of the [Gardener](https://gardener.cloud/) by implementing a logic for dynamically forwarding log messages from one Fluent-bit to multiple Vali instances.
It also adds additional configurations that aim to improve plugin's performance and user experience.

## Configuration Options

| Key           | Description                                   | Default                             |
| --------------|-----------------------------------------------|-------------------------------------|
| Url           | Url of vali server API endpoint.               | `http://localhost:3100/vali/api/v1/push` |
| ProxyURL      | Optional Url for http proxy.                   | ""                                 |
| BatchWait     | Time to wait before send a log batch to Vali, full or not. (unit: sec) | 1 second   |
| BatchSize     | Log batch size to send a log batch to Vali (unit: Bytes).    | 10 KiB (10 * 1024 Bytes) |
| MaxRetries     | Number of times the vali client will try to send unsuccessful sent record to vali.    | 10 |
| Timeout     | The duration which vali client will wait for response.   | 10 |
| MinBackoff     | The first wait after unsuccessful sent log.    | 0.5s |
| MaxBackoff     | The maximum duration after  unsuccessful sent log.  | 5m |
| Labels        | Additional labels in the log stream                       | {job="fluent-bit"}                    |
| LogLevel      | LogLevel for plugin logger.                    | "info"                              |
| RemoveKeys    | Fields to be removed from the log streams                        | none                                |
| AutoKubernetesLabels | If set to true, it will add all Kubernetes labels to Vali labels | false    |
| LabelKeys     | Comma separated list of keys to use as stream labels. All other keys will be placed into the log line. LabelKeys is deactivated when using `LabelMapPath` label mapping configuration. | none |
| LineFormat    | Format to use when flattening the record to a log line. Valid values are "json" or "key_value". If set to "json" the log line sent to Vali will be the `fluent-bit` record (excluding any keys extracted out as labels) dumped as json. If set to "key_value", the log line will be each item in the record concatenated together (separated by a single space) in the format `<key>=<value>`. | json |
| DropSingleKey | When a record has only a single key after after extracting the label keys for the stream, the log line sent to Vali will just be the value of the remaining key.| true |
| LabelMapPath | Path to a json file defining how to transform nested records. | none |
| DynamicHostPath | Jsonpath in the log labels to the dynamic host. | none |
| DynamicHostPrefix | String to prepend to the dynamic host. | none |
| DynamicHostSuffix | String to append to the dynamic host. | none |
| DynamicHostRegex | Regex to check if the dynamic host is valid. | '*' |
| Buffer | If set to true, a buffered client will be used. | none |
| BufferType | The buffer type to use when using buffered client is unable. "Dque" is the only available. | "dque" |
| QueueDir | Path to a directory where the buffer will store its records. | '/tmp/flb-storage/vali' |
| QueueSegmentSize | The number of entries stored into the buffer. | 500 |
| QueueName | The name of the file where the log entries will be stored | `dque` |
| SortByTimestamp | Sort the logs by their timestamps. | `false` |
| FallbackToTagWhenMetadataIsMissing | If set the plugin will try to extract the `namespace`, `pod_name` and `container_name` from the tag when the metadata is missing | `false` |
| TagKey | The key of the record which holds the tag. The tag should not be nested | "tag" |
| TagPrefix | The prefix of the tag. In the prefix no metadata will be searched. The prefix must not contain group expression(`()`). | none |
| TagExpression | The regex expression which will be used for matching the metadata retrieved from the tag. It contains 3 group expressions (`()`): `pod name`, `namespace` and the `container name` | "\\.(.*)_(.*)_(.*)-.*\\.log" |
| DropLogEntryWithoutK8sMetadata | When metadata is missing for the log entry, it will be dropped | `false` |
| ControllerSyncTimeout | Time to wait for cluster object synchronization | 60 seconds |
| NumberOfBatchIDs | The number of id per batch. This increase the number of vali label streams | 10 |
| IdLabelName | The name of the batch ID label kye in the stream label set | `id` |
| DeletedClientTimeExpiration | The time duration after a client for deleted cluster will be considered for expired | 1 hour |
| HostnameKeyValue | \<hostname-kye\>\<space\>\<hostname-value\> key/value pair adding the hostname into the label stream. When value is omitted the hostname is deduced from os.Hostname() call | nil |
| Pprof | Activating the pprof packeg for debugging purpose | false |
| LabelSetInitCapacity | The initial size of the label set which will be extracted from the records. Reduce map reallocation | 10 |
| SendLogsToMainClusterWhenIsInCreationState | Send log to the dynamic cluster when it is in creation state | `true` |
| SendLogsToMainClusterWhenIsInReadyState | Send log to the dynamic cluster when it is in ready state | `true` |
| SendLogsToMainClusterWhenIsInHibernatingState | Send log to the dynamic cluster when it is in hibernating state  | `false` |
| SendLogsToMainClusterWhenIsInHibernatedState | Send log to the dynamic cluster when it is in hibernated state | `false` |
| SendLogsToMainClusterWhenIsInDeletionState | Send log to the dynamic cluster when it is in deletion state | `true` |
| SendLogsToMainClusterWhenIsInRestoreState | Send log to the dynamic cluster when it is in restoration state | `true` |
| SendLogsToMainClusterWhenIsInMigrationState | Send log to the dynamic cluster when it is in migration state | `true` |
| SendLogsToDefaultClientWhenClusterIsInCreationState | Send log to the default URL when it is in creation state | `true` |
| SendLogsToDefaultClientWhenClusterIsInReadyState | Send log to the default URL when it is in ready state | `false` |
| SendLogsToDefaultClientWhenClusterIsInHibernatingState | Send log to the default URL when it is in hibernating state | `false` |
| SendLogsToDefaultClientWhenClusterIsInHibernatedState | Send log to the default URL when it is in hibernated state | `false` |
| SendLogsToDefaultClientWhenClusterIsInDeletionState | Send log to the default URL when it is in deletion state | `true` |
| SendLogsToDefaultClientWhenClusterIsInRestoreState | Send log to the default URL when it is in restoration state | `true` |
| SendLogsToDefaultClientWhenClusterIsInMigrationState | Send log to the default URL when it is in migration state | `true` |

### Labels

Labels are used to query logs. For example, `{container_name="nginx", cluster="us-west1"}`. Usually labels represent metadata about the workload producing the log stream such as (`instance`, `container_name`, `region`, `cluster`, `level`).  In Vali labels are indexed consequently which may lead to log streams with high cardinality. The latter can influence the performance of the backend

The `Labels`, `RemoveKeys` , `LabelKeys` and `LabelMapPath` configuration options determine how the output plugin will perform labels extraction.

### AutoKubernetesLabels

If set to true, it will add all Kubernetes labels to Vali labels automatically and ignore parameters `LabelKeys`, LabelMapPath.

### LabelMaps

While Vali labels are key value pairs, the fluent-bit records may contain nested structures.
The `LabelMap` or `LabelMapPath` determines how to extract `labels` from each record. Each key in the map will be matched with the log record to find the label values. Values from the configuration are used as label names.

Considering the record below :

```json
{
  "kubernetes": {
    "container_name": "valitail",
    "pod_name": "valitail-xxx",
    "namespace_name": "prod",
    "labels" : {
        "team": "x-men",
    },
  },
  "HOSTNAME": "docker-desktop",
  "log" : "a log line",
  "time": "20190926T152206Z",
}
```

and a LabelMap file as follow :

```json
{
  "kubernetes": {
    "container_name": "container",
    "pod_name": "pod",
    "namespace_name": "namespace",
    "labels" : {
        "team": "team",
    },
  },
}
```

The labels extracted will be `{team="x-men", container="valitail", pod="valitail-xxx", namespace="prod"}`.

If you don't want the `kubernetes` and `HOSTNAME` fields to appear in the log line you can use the `RemoveKeys` configuration field. (e.g. `RemoveKeys kubernetes,HOSTNAME`).

### Configuration examples

To configure the Vali output plugin add this section to fluent-bit.conf

```properties
[Output]
    Name gardenervali
    Match kubernetes.*
    Url http://vali.garden.svc:3100/vali/api/v1/push
    LogLevel info
    BatchWait 40
    BatchSize 30720
    Labels {test="fluent-bit-go"}
    LineFormat json
    SortByTimestamp true
    DropSingleKey false
    AutoKubernetesLabels false
    LabelSelector gardener.cloud/role:shoot
    RemoveKeys kubernetes,stream,time,tag
    LabelMapPath /fluent-bit/etc/kubernetes_label_map.json
    DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
    DynamicHostPrefix http://vali.
    DynamicHostSuffix .svc:3100/vali/api/v1/push
    DynamicHostRegex ^shoot-
    DynamicTenant user gardener user
    MaxRetries 3
    Timeout 10
    MinBackoff 30
    Buffer true
    BufferType dque
    QueueDir  /fluent-bit/buffers/operator
    QueueSegmentSize 300
    QueueSync normal
    QueueName gardener-kubernetes-operator
    FallbackToTagWhenMetadataIsMissing true
    TagKey tag
    DropLogEntryWithoutK8sMetadata true
```

```properties
[Output]
    Name gardenervali
    Match journald.*
    Url http://vali.garden.svc:3100/vali/api/v1/push
    LogLevel info
    BatchWait 60
    BatchSize 30720
    Labels {test="fluent-bit-go"}
    LineFormat json
    SortByTimestamp true
    DropSingleKey false
    RemoveKeys kubernetes,stream,hostname,unit
    LabelMapPath /fluent-bit/etc/systemd_label_map.json
    MaxRetries 3
    Timeout 10
    MinBackoff 30
    Buffer true
    BufferType dque
    QueueDir  /fluent-bit/buffers
    QueueSegmentSize 300
    QueueSync normal
    QueueName gardener-journald
```

### Running multiple plugin instances

You can run multiple plugin instances in the same fluent-bit process, for example if you want to push to different Vali servers or route logs into different Vali tenant IDs. To do so, add additional `[Output]` sections.

## Building

```bash
make plugin
```

## Prerequisites

* Go 1.23+
* gcc (for cgo)

## Local

If you have Fluent Bit installed in your `$PATH` you can run the plugin using:

```bash
fluent-bit -e /path/to/built/out_vali.so -c fluent-bit.conf
```

You can also adapt your plugins.conf, removing the need to change the command line options:

```config
[PLUGINS]
    Path /path/to/built/out_vali.so
```
