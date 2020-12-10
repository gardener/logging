# Fluent Bit output plugin

This plugin extends [Grafana,s Fluent Bit output plugin](https://github.com/grafana/loki/tree/v1.6.0/cmd/fluent-bit) which aims to forward log messages from fluent-bit to Loki.
Тhe plugin meets the needs of the [Gardener](https://gardener.cloud/) by implementing a logic for dynamically forwarding log messages from one Fluent-bit to multiple Loki instances.
It also adds additional configurations that aim to improve plugin's performance and user experience.

## Configuration Options

| Key           | Description                                   | Default                             |
| --------------|-----------------------------------------------|-------------------------------------|
| Url           | Url of loki server API endpoint.               | http://localhost:3100/loki/api/v1/push |
| TenantID      | The tenant ID used by default to push logs to Loki. If omitted or empty it assumes Loki is running in single-tenant mode and no `X-Scope-OrgID` header is sent.               | "" |
| BatchWait     | Time to wait before send a log batch to Loki, full or not. (unit: sec) | 1 second   |
| BatchSize     | Log batch size to send a log batch to Loki (unit: Bytes).    | 10 KiB (10 * 1024 Bytes) |
| MaxRetries     | Number of times the loki client will try to send unsuccessful sent record to loki.    | 10 |
| Timeout     | The duration which loki client will wait for response.   | 10 |
| MinBackoff     | The first wait after unsuccessful sent log.    | 0.5s |
| MaxBackoff     | The maximum duration after  unsuccessful sent log.  | 5m |
| Labels        | labels for API requests.                       | {job="fluent-bit"}                    |
| LogLevel      | LogLevel for plugin logger.                    | "info"                              |
| RemoveKeys    | Specify removing keys.                         | none                                |
| AutoKubernetesLabels | If set to true, it will add all Kubernetes labels to Loki labels | false    |
| LabelKeys     | Comma separated list of keys to use as stream labels. All other keys will be placed into the log line. LabelKeys is deactivated when using `LabelMapPath` label mapping configuration. | none |
| LineFormat    | Format to use when flattening the record to a log line. Valid values are "json" or "key_value". If set to "json" the log line sent to Loki will be the fluentd record (excluding any keys extracted out as labels) dumped as json. If set to "key_value", the log line will be each item in the record concatenated together (separated by a single space) in the format <key>=<value>. | json |
| DropSingleKey | If set to true and after extracting label_keys a record only has a single key remaining, the log line sent to Loki will just be the value of the record key.| true |
| LabelMapPath | Path to a json file defining how to transform nested records. | none
| DynamicHostPath | Jsonpath in the log labels to the dynamic host. | none
| DynamicHostPrefix | String to prepend to the dynamic host. | none
| DynamicHostSuffix | String to append to the dynamic host. | none
| DynamicHostRegex | Regex to check if the dynamic host is valid. | '*'
| Buffer | If set to true, a buffered client will be used. | none
| BufferType | The buffer type to use when using buffered client is unable. "Dque" is the only available. | "dque"
| QueueDir | Path to a directory where the buffer will store its records. | '/tmp/flb-storage/loki'
| QueueSegmentSize | The number of entries stored into the buffer. | 500
| QueueName | The name of the file where the log entries will be stored | `dque`
| ReplaceOutOfOrderTS | Overwrites the timestamp of out of order records. Their timestamp will replaced with the timestamp of the last entry. | `false`
| FallbackToTagWhenMetadataIsMissing | If set the plugin will try to extract the `namespace`, `pod_name` and `container_name` from the tag when the metadata is missing | `false`
| TagKey | The key of the record which holds the tag. The tag should not be nested | "tag"
| TagPrefix | The prefix of the tag. In the prefix no metadata will be searched. The prefix must not contain group expression(`()`). | none
| TagExpression | The regex expression which will be used for matching the metadata retrieved from the tag. It contains 3 group expressions (`()`): `pod name`, `namespace` and the `container name` | "\\.(.*)_(.*)_(.*)-.*\\.log"
| DropLogEntryWithoutK8sMetadata | When metadata is missing for the log entry, it will be dropped | `false`


### Labels

Labels are used to [query logs](https://github.com/grafana/loki/blob/v1.5.0/docs/logql.md) `{container_name="nginx", cluster="us-west1"}`, they are usually metadata about the workload producing the log stream (`instance`, `container_name`, `region`, `cluster`, `level`).  In Loki labels are indexed consequently you should be cautious when choosing them (high cardinality label values can have performance drastic impact).

You can use `Labels`, `RemoveKeys` , `LabelKeys` and `LabelMapPath` to how the output plugin will perform labels extraction.

### AutoKubernetesLabels

If set to true, it will add all Kubernetes labels to Loki labels automatically and ignore parameters `LabelKeys`, LabelMapPath.

### LabelMapPath

When using the `Parser` and `Filter` plugins Fluent Bit can extract and add data to the current record/log data. While Loki labels are key value pair, record data can be nested structures.
You can pass a json file that defines how to extract [labels](https://github.com/grafana/loki/blob/v1.5.0/docs/overview/README.md#overview-of-loki) from each record. Each json key from the file will be matched with the log record to find label values. Values from the configuration are used as label names.

Considering the record below :

```json
{
  "kubernetes": {
    "container_name": "promtail",
    "pod_name": "promtail-xxx",
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

The labels extracted will be `{team="x-men", container="promtail", pod="promtail-xxx", namespace="prod"}`.

If you don't want the `kubernetes` and `HOSTNAME` fields to appear in the log line you can use the `RemoveKeys` configuration field. (e.g. `RemoveKeys kubernetes,HOSTNAME`).

### Configuration examples

To configure the Loki output plugin add this section to fluent-bit.conf

```properties
[Output]
    Name gardenerloki
    Match kubernetes.*
    Url http://loki.garden.svc:3100/loki/api/v1/push
    LogLevel info
    BatchWait 40
    BatchSize 30720
    Labels {test="fluent-bit-go"}
    LineFormat json
    ReplaceOutOfOrderTS true
    DropSingleKey false
    AutoKubernetesLabels false
    LabelSelector gardener.cloud/role:shoot
    RemoveKeys kubernetes,stream,time,tag
    LabelMapPath /fluent-bit/etc/kubernetes_label_map.json
    DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
    DynamicHostPrefix http://loki.
    DynamicHostSuffix .svc:3100/loki/api/v1/push
    DynamicHostRegex ^shoot-
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
    TenantID operator
```

```properties
[Output]
    Name gardenerloki
    Match {{ .Values.exposedComponentsTagPrefix }}.kubernetes.*
    Url http://loki.garden.svc:3100/loki/api/v1/push
    LogLevel info
    BatchWait 40
    BatchSize 30720
    Labels {test="fluent-bit-go", lang="Golang"}
    LineFormat json
    ReplaceOutOfOrderTS true
    DropSingleKey false
    AutoKubernetesLabels true
    LabelSelector gardener.cloud/role:shoot
    RemoveKeys kubernetes,stream,type,time,tag
    LabelMapPath /fluent-bit/etc/kubernetes_label_map.json
    DynamicHostPath {"kubernetes": {"namespace_name": "namespace"}}
    DynamicHostPrefix http://loki.
    DynamicHostSuffix .svc:3100/loki/api/v1/push
    DynamicHostRegex ^shoot-
    MaxRetries 3
    Timeout 10
    MinBackoff 30
    Buffer true
    BufferType dque
    QueueDir  /fluent-bit/buffers/user
    QueueSegmentSize 300
    QueueSync normal
    QueueName gardener-kubernetes-user
    FallbackToTagWhenMetadataIsMissing true
    TagKey tag
    DropLogEntryWithoutK8sMetadata true
    TenantID user
```

```properties
[Output]
    Name gardenerloki
    Match journald.*
    Url http://loki.garden.svc:3100/loki/api/v1/push
    LogLevel info
    BatchWait 60
    BatchSize 30720
    Labels {test="fluent-bit-go"}
    LineFormat json
    ReplaceOutOfOrderTS true
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
A full [example configuration file](example.config) is also available in this repository.

### Running multiple plugin instances

You can run multiple plugin instances in the same fluent-bit process, for example if you want to push to different Loki servers or route logs into different Loki tenant IDs. To do so, add additional `[Output]` sections.

## Building

```bash
make plugin
```

## Prerequisites

* Go 1.11+
* gcc (for cgo)


## Local

If you have Fluent Bit installed in your `$PATH` you can run the plugin using:

```bash
fluent-bit -e /path/to/built/out_loki.so -c fluent-bit.conf
```

You can also adapt your plugins.conf, removing the need to change the command line options:

```
[PLUGINS]
    Path /path/to/built/out_loki.so
```