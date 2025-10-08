#!/usr/bin/env bash

set -eo pipefail
dir=$(dirname "$0")


function create_namespaces {
    local shoot_namespace="shoot--logging--dev-${1}"

    kubectl create namespace "$shoot_namespace" \
      --dry-run=client -o yaml | \
      kubectl apply -f -
}

function create_services {
    local shoot_namespace="shoot--logging--dev-${1}"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: logging
  namespace: "$shoot_namespace"
spec:
  type: ExternalName
  externalName: logging-vali-shoot.fluent-bit.svc.cluster.local
EOF
}

function create_clusters {
    local cluster_name="shoot--logging--dev-${1}"
    local shoot_namespace="shoot--logging--dev-${1}"
    local shoot_name="dev-${1}"
    local project_namespace="logging"


    yq e ".metadata.name = \"$cluster_name\" | .spec.shoot.metadata.name = \"$shoot_name\" | .spec.shoot.metadata.namespace = \"$project_namespace\""  $dir/cluster/cluster.yaml | \
    kubectl apply -f -
}

function create_jobs {
    local shoot_namespace="shoot--logging--dev-${1}"

    cat <<EOF | kubectl apply -f -
kind: Job
apiVersion: batch/v1
metadata:
  name: logger
  namespace: "$shoot_namespace"
spec:
  parallelism: $JOBS
  completions: $JOBS
  template:
    spec:
      containers:
      - name: logger
        image: nickytd/log-generator:latest
        args:
          - --wait=${LOGS_DELAY}
          - --count=${LOGS}
      restartPolicy: Never
EOF
}

for ((i=1; i<=CLUSTERS; i++)); do
    create_namespaces "$i" &
done
wait

for ((i=1; i<=CLUSTERS; i++)); do
    create_services "$i" &
done
wait


for ((i=1; i<=CLUSTERS; i++)); do
    create_clusters "$i" &
done
wait

for ((i=1; i<=CLUSTERS; i++)); do
  create_jobs "$i" &
done
wait


echo "Generated $CLUSTERS clusters"
