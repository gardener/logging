# logging

[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/logging)](https://goreportcard.com/report/github.com/gardener/logging)

This repository contains components needed for Gardener logging.

## What's inside

[Fluentd for Elasticsearch](fluentd-es) - a Docker image with Fluentd for Elasticsearch - based on the original provided by Kubernetes (https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/fluentd-elasticsearch/fluentd-es-image) with some customizations for Garderner.

[Curator for Elasticsearch](curator-es) - a Docker image with Curator (https://github.com/elastic/curator).

## Local build

```bash
$ make docker-images
```
