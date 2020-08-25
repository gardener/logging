# logging

[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/logging)](https://goreportcard.com/report/github.com/gardener/logging)

This repository contains components needed for Gardener logging.

## What's inside

[Fluent-bit with outplugin to loki](fluent-bit-to-loki) - a Docker image with [Fluent-bit](https://github.com/fluent/fluent-bit) containing pre-build golang plugin
for sending logs to multiple Loki instances.
## Local build

```bash
$ make docker-images
```
