#!/bin/bash

set -o nounset #catch and prevent errors caused by the use of unset variables.
set -o pipefail #exit with the exit code of the first error
set -o errexit #exits immediately if any command in a script exits with a non-zero status

function __catch() {
  local cmd="${1:-}"
  echo
  echo "errexit $cmd on line $(caller)" >&2
}
trap '__catch "${BASH_COMMAND}"' ERR


repo_root="$(readlink -f $(dirname ${0})/..)"
echo "REPO_ROOT ${repo_root}"
if [[ ! -d "$repo_root/gardener" ]]; then
  # use gardener/gardener v1.71.2
  echo "fetch https://github.com/gardener/gardener.git v1.71.2"
  git clone --depth 1 --branch v1.71.2 https://github.com/gardener/gardener.git > /dev/null 2>&1
fi

# Start Kind cluster
cd "$repo_root/gardener"
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
# docker tag eu.gcr.io/gardener-project/gardener/fluent-bit-to-vali:latest fluent-bit-to-vali:$version
# docker tag eu.gcr.io/gardener-project/gardener/vali-curator:latest       vali-curator:$version
docker tag eu.gcr.io/gardener-project/gardener/telegraf-iptables:latest  telegraf-iptables:$version
docker tag eu.gcr.io/gardener-project/gardener/tune2fs:latest            tune2fs:$version
docker tag eu.gcr.io/gardener-project/gardener/event-logger:latest       event-logger:$version

#Load the images into the Kind cluster.
# kind load docker-image fluent-bit-to-vali:$version --name gardener-local
# kind load docker-image vali-curator:$version       --name gardener-local
kind load docker-image telegraf-iptables:$version  --name gardener-local
kind load docker-image tune2fs:$version  --name gardener-local
kind load docker-image event-logger:$version       --name gardener-local

# Change the image of the loggings in the gardener repo
cd "$repo_root/gardener"
yq -i e "(.images[] | select(.name == \"event-logger\") | .repository) |= \"docker.io/library/event-logger\"" "charts/images.yaml"
yq -i e "(.images[] | select(.name == \"event-logger\") | .tag) |= \"$version\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .repository) |= \"docker.io/library/fluent-bit-to-vali\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .tag) |= \"$version\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"vali-curator\") | .repository) |= \"docker.io/library/vali-curator\"" "charts/images.yaml"
# yq -i e "(.images[] | select(.name == \"vali-curator\") | .tag) |= \"$version\"" "charts/images.yaml"
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

GO111MODULE=on $repo_root/tools/ginkgo run --timeout=1h --v --show-node-events --progress "$@"

cd "$repo_root/gardener"
make gardener-down
