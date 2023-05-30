#!/usr/bin/env bash

dir="$(dirname "$0")"

set -o nounset #catch and prevent errors caused by the use of unset variables.
set -o pipefail #exit with the exit code of the first error
set -o errexit #exits immediately if any command in a script exits with a non-zero status

source "$dir/.includes.sh"

# Make local images with uniq tags
version=$(git rev-parse HEAD) # Get the hash of the current commit
docker tag eu.gcr.io/gardener-project/gardener/fluent-bit-to-vali:latest fluent-bit-to-vali:$version
docker tag eu.gcr.io/gardener-project/gardener/vali-curator:latest       vali-curator:$version
docker tag eu.gcr.io/gardener-project/gardener/telegraf-iptables:latest  telegraf-iptables:$version
docker tag eu.gcr.io/gardener-project/gardener/tune2fs:latest            tune2fs:$version
docker tag eu.gcr.io/gardener-project/gardener/event-logger:latest       event-logger:$version

# Load the images into the Kind cluster.
$repo_root/tools/kind load docker-image fluent-bit-to-vali:$version --name gardener-local
$repo_root/tools/kind load docker-image vali-curator:$version       --name gardener-local
$repo_root/tools/kind load docker-image telegraf-iptables:$version  --name gardener-local
$repo_root/tools/kind load docker-image tune2fs:$version  --name gardener-local
$repo_root/tools/kind load docker-image event-logger:$version       --name gardener-local

# Check to see if the gardenr repo is already fetched
[[ ! -d "$repo_root/gardener" ]] && echo "fetch gardener repo in $repo_root"

# Change the image of the loggings in the fetched gardener repo
target="$repo_root/gardener/charts/images.yaml"
$repo_root/tools/yq -i e "(.images[] | select(.name == \"event-logger\") | .repository) |= \"docker.io/library/event-logger\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"event-logger\") | .tag) |= \"$version\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .repository) |= \"docker.io/library/fluent-bit-to-vali\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .tag) |= \"$version\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"vali-curator\") | .repository) |= \"docker.io/library/vali-curator\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"vali-curator\") | .tag) |= \"$version\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"telegraf\") | .repository) |= \"docker.io/library/telegraf-iptables\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"telegraf\") | .tag) |= \"$version\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"tune2fs\") | .repository) |= \"docker.io/library/tune2fs\"" $target
$repo_root/tools/yq -i e "(.images[] | select(.name == \"tune2fs\") | .tag) |= \"$version\"" $target