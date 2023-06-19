#!/usr/bin/env bash

# Check if a kind cluster with a specific name is running
function __check_kind_cluster() {
  local kind_name="$1"

  running_clusters=$(kind get clusters 2>/dev/null)
  [[ "$running_clusters" == *"$kind_name"* ]] && echo "exists"
}

function __check_container_image() {
  local container_image="$1"

  docker image inspect $container_image > /dev/null 2>&1
  [[ "$?" -eq "0" ]] && echo "exists"
}

dir="$(dirname "$0")"
source "$dir/.includes.sh"
images=(fluent-bit-to-vali \
        event-logger \
        vali-curator \
        telegraf-iptables \
        tune2fs)

# Make local images with uniq tags
version=$(git rev-parse HEAD) # Get the hash of the current commit

# Change the image of the loggings in the fetched gardener repo
target="$repo_root/gardener/charts/images.yaml"

# Fetch the gardener repo
[[ ! -d "$repo_root/gardener" ]] && "$dir/fetch-gardener.sh"

for img in "${images[@]}"; do
  # tag "latest "container image if exists
  container_image="eu.gcr.io/gardener-project/gardener/${img}:latest"
  if [[ "exists" == $(__check_container_image ${container_image}) ]]; then
    docker tag ${container_image} ${img}:${version}
  else
    echo "container image ${container_image} is not found" || continue
  fi

  # update gardenrer chart images.yaml
  $dir/update_chart_images.sh "docker.io/library/${img}" "${version}"

  # load container image in the kind cluster
  if [[ "exists" == $(__check_kind_cluster "gardener-local") ]]; then
     $repo_root/tools/kind load docker-image ${img}:$version --name gardener-local
  else
      echo "gardnere-local kind cluster is not available"
  fi
done
