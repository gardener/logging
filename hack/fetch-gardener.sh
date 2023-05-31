#!/usr/bin/env bash

dir="$(dirname "$0")"

set -o errexit #exits immediately if any command in a script exits with a non-zero status

source "$dir/.includes.sh"

echo "REPO_ROOT ${repo_root}"
if [[ ! -d "$repo_root/gardener" ]]; then
  # use gardener/gardener version defined at .includes.sh ${gardener}
  echo "fetch https://github.com/gardener/gardener.git ${gardener}"
  git clone --depth 1 --branch ${gardener} https://github.com/gardener/gardener.git > /dev/null 2>&1
fi
