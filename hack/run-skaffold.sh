#!/usr/bin/env bash

# target cluster
KUBECONFIG="$(dirname $0)/../gardener/example/gardener-local/kind/local/kubeconfig"

# set kubeconfig if needed
kubeconfig="--kubeconfig="${KUBECONFIG}""
for arg in "$@"; do
  [[ "render" == "$arg" ]] && kubeconfig=
done

# run skaffold
LD_FLAGS=`$(dirname $0)/get-build-ld-flags.sh` skaffold $kubeconfig "$@"