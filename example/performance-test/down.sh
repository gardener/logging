#!/usr/bin/env bash

set -eo pipefail
dir=$(dirname "$0")


function delete_jobs {
    local shoot_namespace="shoot--logging--dev-${1}"
    kubectl delete jobs --all -n "$shoot_namespace" --ignore-not-found
}
function delete_services {
    local shoot_namespace="shoot--logging--dev-${1}"
    kubectl delete service --all -n "$shoot_namespace" --ignore-not-found
}

function delete_clusters {
    local cluster_name="shoot--logging--dev-${1}"
    kubectl delete cluster "$cluster_name" --ignore-not-found
}
function delete_namespaces {
    local shoot_namespace="shoot--logging--dev-${1}"
    kubectl delete namespace "$shoot_namespace" --ignore-not-found
}


for ((i=1; i<=CLUSTERS; i++)); do
    delete_jobs "$i" &
done
wait

for ((i=1; i<=CLUSTERS; i++)); do
    delete_services "$i" &
done
wait

for ((i=1; i<=CLUSTERS; i++)); do
    delete_clusters "$i" &
done
wait

for ((i=1; i<=CLUSTERS; i++)); do
    delete_namespaces "$i" &
done
wait

echo "Deleted $CLUSTERS clusters"
