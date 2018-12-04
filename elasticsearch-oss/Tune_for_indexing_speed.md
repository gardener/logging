# Tune for indexing speed
## Use bulk requests
Bulk requests will yield much better performance than single-document index requests. In order to know the optimal size of a bulk request, you should run a benchmark on a single node with a single shard. First try to index 100 documents at once, then 200, then 400, etc. doubling the number of documents in a bulk request in every benchmark run. When the indexing speed starts to plateau then you know you reached the optimal size of a bulk request for your data. In case of tie, it is better to err in the direction of too few rather than too many documents. Beware that too large bulk requests might put the cluster under memory pressure when many of them are sent concurrently, so it is advisable to avoid going beyond a couple tens of megabytes per request even if larger requests seem to perform better.

In our case fluend use the bulk API but the request are limited to 50MB without performin eny tests.
About the memory pressure issue according to the load test the capacity of the ES cluster is about 53kb/s. So If we assume that one shoot control plane resides in one seed node and generates 53k/s that means that the fluentd will send chunks maximum  of 9,54MB. So why our limir is set to 50MB?

## Use multiple workers/threads to send data to ES
A single thread sending bulk requests is unlikely to be able to max out the indexing capacity of an Elasticsearch cluster. In order to use all resources of the cluster, you should send data from multiple threads or processes. In addition to making better use of the resources of the cluster, this should help reduce the cost of each fsync.

Make sure to watch for TOO_MANY_REQUESTS (429) response codes (EsRejectedExecutionException with the Java client), which is the way that Elasticsearch tells you that it cannot keep up with the current indexing rate. When it happens, you should pause indexing a bit before trying again, ideally with randomized exponential backoff.

Similarly to sizing bulk requests, only testing can tell what the optimal number of workers is. This can be tested by progressively increasing the number of workers until either I/O or CPU is saturated on the cluster.

We use multiple thread but for all ES cluster so is more likely to have one thread per ES cluster that multiple but even that way we reach TOO_MANY_REQUESTS (429).
Each thread sends data to ES every 3 minutes.
We can not use exponential backoff because this rule apply even for the new requests for all of the ES cluster.
Instead we give a time of 75 seconds until the next request is sent.

## Increase the refresh interval
The default index.refresh_interval is 1s, which forces Elasticsearch to create a new segment every second. Increasing this value (to say, 30s) will allow larger segments to flush and decreases future merge pressure.

In our case we use curator to set the index.refresh_interval to 60 which reflect as an increased log latency
That is, each recor stored on ES in this period of time can't be seen in the ES until the time has elapsed.

## Disable refresh and replicas for initial loads
If you need to load a large amount of data at once, you should disable refresh by setting index.refresh_interval to -1 and set index.number_of_replicas to 0. This will temporarily put your index at risk since the loss of any shard will cause data loss, but at the same time indexing will be faster since documents will be indexed only once. Once the initial loading is finished, you can set index.refresh_interval and index.number_of_replicas back to their original values.

I could not find any setting how to do that with ealsticsearch plugin of fluentd.

## Disable swapping
You should make sure that the operating system is not swapping out the java process by disabling swapping.

There is no swapping in kubernetes.

## Give memory to the filesystem cache
The filesystem cache will be used in order to buffer I/O operations. You should make sure to give at least half the memory of the machine running Elasticsearch to the filesystem cache.

I have no idea how to do that. In the moment there is about 500MB of memory which can be used by other processes.

## Use faster hardware
If indexing is I/O bound, you should investigate giving more memory to the filesystem cache (see above) or buying faster drives. In particular SSD drives are known to perform better than spinning disks. Always use local storage, remote filesystems such as NFS or SMB should be avoided. Also beware of virtualized storage such as Amazon’s Elastic Block Storage. Virtualized storage works very well with Elasticsearch, and it is appealing since it is so fast and simple to set up, but it is also unfortunately inherently slower on an ongoing basis when compared to dedicated local storage. If you put an index on EBS, be sure to use provisioned IOPS otherwise operations could be quickly throttled.

Stripe your index across multiple SSDs by configuring a RAID 0 array. Remember that it will increase the risk of failure since the failure of any one SSD destroys the index. However this is typically the right tradeoff to make: optimize single shards for maximum performance, and then add replicas across different nodes so there’s redundancy for any node failures. You can also use snapshot and restore to backup the index for further insurance.

TODO: Lear how to spawn ES pod in nodes which are using SSD

## Indexing buffer size
If your node is doing only heavy indexing, be sure indices.memory.index_buffer_size is large enough to give at most 512 MB indexing buffer per shard doing heavy indexing (beyond that indexing performance does not typically improve). Elasticsearch takes that setting (a percentage of the java heap or an absolute byte-size), and uses it as a shared buffer across all active shards. Very active shards will naturally use this buffer more than shards that are performing lightweight indexing.

The default is 10% which is often plenty: for example, if you give the JVM 10GB of memory, it will give 1GB to the index buffer, which is enough to host two shards that are heavily indexing.

In our case we set indices.memory.index_buffer_size to 30% which gives about 540MB of memory for the indexing buffer but it gives more to the central ES which can scale verticaly. TODO: fix that by setting the value to 512MB.

## Disable _field_names
The _field_names field introduces some index-time overhead, so you might want to disable it if you never need to run exists queries.
Disabling _field_names is often not necessary because it no longer carries the index overhead it once did.

Have no Idea what is that and do we need exist querrys in future. TODO: find how to disable that option by the configuration files.

## Additional optimizations
Many of the strategies outlined in Tune for disk usage also provide an improvement in the speed of indexing.

### Disable the features you do not need
1. By default Elasticsearch indexes and adds doc values to most fields so that they can be searched and aggregated out of the box. For instance if you have a numeric field called "foo" that you need to run histograms on but that you never need to filter on, you can safely disable indexing on this field in your mappings:
```
PUT index
{
  "mappings": {
    "_doc": {
      "properties": {
        "foo": {
          "type": "integer",
          "index": false
        }
      }
    }
  }
}
```
**TODO: do we have such indices?**

2. **text** fields store normalization factors in the index in order to be able to score documents. If you only need matching capabilities on a text field but do not care about the produced scores, you can configure Elasticsearch to not write norms to the index:
```
PUT index
{
  "mappings": {
    "_doc": {
      "properties": {
        "foo": {
          "type": "text",
          "norms": false
        }
      }
    }
  }
}
```
**TODO: do we need scores?**

3. **text** fields also store frequencies and positions in the index by default. Frequencies are used to compute scores and positions are used to run phrase queries. If you do not need to run phrase queries, you can tell Elasticsearch to not index positions:
```
PUT index
{
  "mappings": {
    "_doc": {
      "properties": {
        "foo": {
          "type": "text",
          "index_options": "freqs"
        }
      }
    }
  }
}
```

**TODO: do we need phrase queries?**

4. Furthermore if you do not care about scoring either, you can configure Elasticsearch to just index matching documents for every term. You will still be able to search on this field, but phrase queries will raise errors and scoring will assume that terms appear only once in every document.
```
PUT index
{
  "mappings": {
    "_doc": {
      "properties": {
        "foo": {
          "type": "text",
          "norms": false,
          "index_options": "freqs"
        }
      }
    }
  }
}

```
**TODO: do we need scores and the number of term appearences?**

