#!/usr/bin/env bash

dir="$(dirname "$0")"

set -o nounset #catch and prevent errors caused by the use of unset variables.
set -o pipefail #exit with the exit code of the first error
set -o errexit #exits immediately if any command in a script exits with a non-zero status

source "$dir/.includes.sh"

echo "REPO_ROOT ${repo_root}"
if [[ ! -d "$repo_root/gardener" ]]; then
  # use gardener/gardener version defined at .includes.sh ${gardener}
  echo "fetch https://github.com/gardener/gardener.git ${gardener}"
  git clone --depth 1 --branch ${gardener} https://github.com/gardener/gardener.git > /dev/null 2>&1
fi

# Start Kind cluster
make -C "$repo_root/gardener" kind-down # in case the test is run twice somehow skipping trap
make -C "$repo_root/gardener" kind-up

trap '{  
  make -C "$repo_root/gardener" kind-down
}' EXIT

# make shoot domains accessible to test
if ! grep -q "127.0.0.1 api.local.local.external.local.gardener.cloud" /etc/hosts; then
  echo 127.0.0.1 api.local.local.external.local.gardener.cloud  >> /etc/hosts
fi

if ! grep -q "127.0.0.1 api.local.local.internal.local.gardener.cloud" /etc/hosts; then
  echo 127.0.0.1 api.local.local.internal.local.gardener.cloud   >> /etc/hosts
fi

# Build docker images
make -C $repo_root docker-images

# Load container images in the gardener-local kind cluster
source $dir/load-container-images.sh

export KUBECONFIG=$repo_root/gardener/example/gardener-local/kind/local/kubeconfig
make -C $repo_root/gardener gardener-up

# reduce flakiness in contended pipelines
export GOMEGA_DEFAULT_EVENTUALLY_TIMEOUT=5s
export GOMEGA_DEFAULT_EVENTUALLY_POLLING_INTERVAL=200ms
# if we're running low on resources, it might take longer for tested code to do something "wrong"
# poll for 5s to make sure, we're not missing any wrong action
export GOMEGA_DEFAULT_CONSISTENTLY_DURATION=5s
export GOMEGA_DEFAULT_CONSISTENTLY_POLLING_INTERVAL=200ms

GO111MODULE=on $repo_root/tools/ginkgo run --timeout=1h --v --show-node-events --progress "$@"

make -C "$repo_root/gardener" gardener-down
