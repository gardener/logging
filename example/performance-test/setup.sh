#!/usr/bin/env bash

set -e

dir=$(dirname "$0")

kubectl apply -f ${dir}/cluster/cluster-crd.yaml
namespace=fluent-bit

helm upgrade logging \
  ${dir}/charts/fluent-bit-plugin \
  --values=${dir}/values.yaml \
  --install \
  --namespace $namespace \
  --create-namespace \
  --timeout 300s