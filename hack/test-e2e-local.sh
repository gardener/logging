#!/bin/bash


set -o nounset #catch and prevent errors caused by the use of unset variables.
set -o pipefail #exit with the exit code of the first error
set -o errexit #exits immediately if any command in a script exits with a non-zero status

repo_root="$(readlink -f $(dirname ${0})/..)"
echo "REPO_ROOT ${repo_root}"
if [[ ! -d "$repo_root/gardener" ]]; then
  git clone https://github.com/gardener/gardener.git
fi

# Start Kind cluster
cd "$repo_root/gardener"
git checkout 1a7052b51f76c8f317be3dd8d91b420afdb611f3 # g/g v1.69.1

make kind-down # in case the test is run twice somehow skipping trap
make kind-up

trap '{
  cd "$repo_root/gardener"
  make kind-down
}' EXIT

# make shoot domains accessible to test
if ! grep -q "127.0.0.1 api.local.local.external.local.gardener.cloud" /etc/hosts; then
  echo 127.0.0.1 api.local.local.external.local.gardener.cloud  >> /etc/hosts
fi

if ! grep -q "127.0.0.1 api.local.local.internal.local.gardener.cloud" /etc/hosts; then
  echo 127.0.0.1 api.local.local.internal.local.gardener.cloud   >> /etc/hosts
fi





# Build docker images
cd $repo_root
make docker-images

# # Make local images with uniq tags 
version=$(git rev-parse HEAD) # Get the hash of the current commit
# docker tag eu.gcr.io/gardener-project/gardener/fluent-bit-to-loki:latest fluent-bit-to-loki:$version
# docker tag eu.gcr.io/gardener-project/gardener/loki-curator:latest       loki-curator:$version
docker tag eu.gcr.io/gardener-project/gardener/telegraf-iptables:latest  telegraf-iptables:$version
docker tag eu.gcr.io/gardener-project/gardener/tune2fs:latest            tune2fs:$version
docker tag eu.gcr.io/gardener-project/gardener/event-logger:latest       event-logger:$version

#Load the images into the Kind cluster.
# kind load docker-image fluent-bit-to-loki:$version --name gardener-local
# kind load docker-image loki-curator:$version       --name gardener-local
kind load docker-image telegraf-iptables:$version  --name gardener-local
kind load docker-image tune2fs:$version  --name gardener-local
kind load docker-image event-logger:$version       --name gardener-local

# Change the image of the loggings in the gardener repo
cd "$repo_root/gardener"
yq -i e "(.images[] | select(.name == \"event-logger\") | .repository) |= \"docker.io/library/event-logger\"" "charts/images.yaml"
yq -i e "(.images[] | select(.name == \"event-logger\") | .tag) |= \"$version\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .repository) |= \"docker.io/library/fluent-bit-to-loki\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .tag) |= \"$version\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"loki-curator\") | .repository) |= \"docker.io/library/loki-curator\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"loki-curator\") | .tag) |= \"$version\"" "charts/images.yaml"
yq -i e "(.images[] | select(.name == \"telegraf\") | .repository) |= \"docker.io/library/telegraf-iptables\"" "charts/images.yaml"
yq -i e "(.images[] | select(.name == \"telegraf\") | .tag) |= \"$version\"" "charts/images.yaml"
yq -i e "(.images[] | select(.name == \"tune2fs\") | .repository) |= \"docker.io/library/tune2fs\"" "charts/images.yaml"
yq -i e "(.images[] | select(.name == \"tune2fs\") | .tag) |= \"$version\"" "charts/images.yaml"


export KUBECONFIG=$repo_root/gardener/example/gardener-local/kind/local/kubeconfig
make gardener-up

cd $repo_root

# reduce flakiness in contended pipelines
export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=5s
export GOMEGA_DEFAULT_EVENTUALLY_POLLING_INTERVAL=200ms
# if we're running low on resources, it might take longer for tested code to do something "wrong"
# poll for 5s to make sure, we're not missing any wrong action
export GOMEGA_DEFAULT_CONSISTENTLY_DURATION=5s
export GOMEGA_DEFAULT_CONSISTENTLY_POLLING_INTERVAL=200ms

GO111MODULE=on ginkgo run --timeout=1h --v --show-node-events --progress "$@"

cd "$repo_root/gardener"
make gardener-down
