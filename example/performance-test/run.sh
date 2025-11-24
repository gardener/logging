#!/usr/bin/env bash

set -eo pipefail
dir=$(dirname "$0")

namespace="fluent-bit"
nameOverride="logging"


function create_namespaces {
    local namespace="${1}"

    kubectl create namespace "$namespace" \
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
    local namespace="${1}"
    kubectl delete job logger --namespace "$namespace" --ignore-not-found >/dev/null 2>&1
    cat <<EOF | kubectl apply -f -
kind: Job
apiVersion: batch/v1
metadata:
  name: logger
  namespace: "$namespace"
spec:
  parallelism: $JOBS
  completions: $JOBS
  ttlSecondsAfterFinished: 600
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

kubectl wait \
  --for=jsonpath='{.status.readyReplicas}'=1 \
  --timeout=300s \
  --namespace ${namespace} \
  statefulset/${nameOverride}-vali-shoot

function shoot {
    local clusters=${CLUSTERS:-10}
    for ((i=1; i<=clusters; i++)); do
        create_namespaces "shoot--logging--dev-${i}" &
    done
    wait

    for ((i=1; i<=clusters; i++)); do
        create_services "$i" &
    done
    wait


    for ((i=1; i<=clusters; i++)); do
        create_clusters "$i" &
    done
    wait

    for ((i=1; i<=clusters; i++)); do
      create_jobs "shoot--logging--dev-${i}" &
    done
    wait
    echo "Generated clusters clusters"
}

function seed {
      local namespace="seed--logging--dev"
      create_namespaces "$namespace"
      create_jobs $namespace
      echo "Generated seed cluster"
}

scenario="${1:-shoot}"
if [ "$scenario" == "shoot" ]; then
    shoot
elif [ "$scenario" == "seed" ]; then
    seed
else
    echo "Unknown scenario: $scenario"
    exit 1
fi
