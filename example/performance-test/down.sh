#!/usr/bin/env bash

set -eo pipefail
dir=$(dirname "$0")


function delete_jobs {
    local namespace="${1}"
    kubectl delete jobs --all -n "$namespace" --ignore-not-found
}
function delete_services {
    local namespace="${1}"
    kubectl delete service --all -n "$namespace" --ignore-not-found
}

function delete_clusters {
    local cluster_name="${1}"
    kubectl delete cluster "$cluster_name" --ignore-not-found
}
function delete_namespaces {
    local namespace="${1}"
    kubectl delete namespace "$namespace" --ignore-not-found
}

function shoot {
    for ((i=1; i<=CLUSTERS; i++)); do
        delete_jobs "shoot--logging--dev-${i}" &
    done
    wait

    for ((i=1; i<=CLUSTERS; i++)); do
        delete_services "shoot--logging--dev-${i}" &
    done
    wait

    for ((i=1; i<=CLUSTERS; i++)); do
        delete_clusters "shoot--logging--dev-${i}" &
    done
    wait

    for ((i=1; i<=CLUSTERS; i++)); do
        delete_namespaces "shoot--logging--dev-${i}" &
    done
    wait

    echo "Deleted $CLUSTERS clusters"
}

function seed {
  local namespace="seed--logging--dev"
  delete_jobs $namespace
  delete_namespaces $namespace
}

scenario="${1:-shoot}"
if [ "$scenario" == "shoot" ]; then
    shoot
elif [ "$scenario" == "seed" ]; then
    seed
else
    echo "Unknown scenario: $scenario"
    exit 1
    echo "Unknown scenario: $scenario"
    exit 1
fi
