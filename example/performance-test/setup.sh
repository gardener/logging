#!/usr/bin/env bash

set -e

dir=$(dirname "$0")

kubectl apply -f ${dir}/cluster/cluster-crd.yaml
namespace=fluent-bit

values_arg=""
if [ -f "${dir}/values.yaml" ]; then
  values_arg="--values=${dir}/values.yaml"
fi

helm upgrade logging \
  ${dir}/charts/fluent-bit-plugin \
  $values_arg \
  --install \
  --namespace $namespace \
  --create-namespace \
  --timeout 300s