#!/usr/bin/env bash

dir="$(dirname "$0")"

source "$dir/.includes.sh"

# Change the image of the loggings in the fetched gardener repo
target="$repo_root/gardener/charts/images.yaml"

fluent_regex="^.*fluent-bit-to-vali$"
function __update_fluent_skaffold_images() {
  local registry="${1:-$(echo "localhost:5001/europe_docker_pkg_dev_gardener-project_gardener/fluent-bit-to-vali")}"
  local version="${2:-$(git rev-parse HEAD 2>/dev/null || echo "latest")}"

  $repo_root/tools/yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .repository) |=\"${registry}\"" $target
  $repo_root/tools/yq -i e "(.images[] | select(.name == \"fluent-bit-plugin-installer\") | .tag) |= \"$version\"" $target
}

event_regex="^.*event-logger$"
function __update_event_skaffold_images() {
  local registry="${1:-$(echo "localhost:5001/europe_docker_pkg_dev_gardener-project_gardener/event-logger")}"
  local version="${2:-$(git rev-parse HEAD 2>/dev/null || echo "latest")}"

  $repo_root/tools/yq -i e "(.images[] | select(.name == \"event-logger\") | .repository) |=\"${registry}\"" $target
  $repo_root/tools/yq -i e "(.images[] | select(.name == \"event-logger\") | .tag) |= \"$version\"" $target
}

curator_regex="^.*vali-curator$"
function __update_curator_skaffold_images() {
  local registry="${1:-$(echo "localhost:5001/europe_docker_pkg_dev_gardener-project_gardener/vali-curator")}"
  local version="${2:-$(git rev-parse HEAD 2>/dev/null || echo "latest")}"

  $repo_root/tools/yq -i e "(.images[] | select(.name == \"vali-curator\") | .repository) |=\"${registry}\"" $target
  $repo_root/tools/yq -i e "(.images[] | select(.name == \"vali-curator\") | .tag) |= \"$version\"" $target
}

telegraf_regex="^.*telegraf-iptables$"
function __update_telegraf_skaffold_images() {
  local registry="${1:-$(echo "localhost:5001/europe_docker_pkg_dev_gardener-project_gardener/telegraf-iptables")}"
  local version="${2:-$(git rev-parse HEAD 2>/dev/null || echo "latest")}"

  $repo_root/tools/yq -i e "(.images[] | select(.name == \"telegraf\") | .repository) |=\"${registry}\"" $target
  $repo_root/tools/yq -i e "(.images[] | select(.name == \"telegraf\") | .tag) |= \"$version\"" $target
}

tune2fs_regex="^.*tune2fs$"
function __update_tune2fs_skaffold_images() {
  local registry="${1:-$(echo "localhost:5001/europe_docker_pkg_dev_gardener-project_gardener/tune2fs")}"
  local version="${2:-$(git rev-parse HEAD 2>/dev/null || echo "latest")}"

  $repo_root/tools/yq -i e "(.images[] | select(.name == \"tune2fs\") | .repository) |=\"${registry}\"" $target
  $repo_root/tools/yq -i e "(.images[] | select(.name == \"tune2fs\") | .tag) |= \"$version\"" $target
}


if [[ ! -z "$1" ]] && [[ "$1" =~ $fluent_regex ]]; then
  __update_fluent_skaffold_images "$@"
  exit 0
fi

if [[ ! -z "$1" ]] && [[ "$1" =~ $event_regex ]]; then
  __update_event_skaffold_images "$@"
  exit 0
fi

if [[ ! -z "$1" ]] && [[ "$1" =~ $curator_regex ]]; then
  __update_curator_skaffold_images "$@"
  exit 0
fi

if [[ ! -z "$1" ]] && [[ "$1" =~ $telegraf_regex ]]; then
  __update_telegraf_skaffold_images "$@"
  exit 0
fi

if [[ ! -z "$1" ]] && [[ "$1" =~ $tune2fs_regex ]]; then
  __update_tune2fs_skaffold_images "$@"
  exit 0
fi
