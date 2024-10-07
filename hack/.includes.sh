#!/usr/bin/env bash

set -o errexit

repo_root="$(readlink -f $(dirname ${0})/..)"
tmp_dir="$( mktemp -d -t tmp-XXXXXX )"

function __cleanup {
  local res=$1
  rm -rf ${res}
}

function __catch() {
  local cmd="${1:-}"
  echo "errexit $cmd on line $(caller)" >&2
}

trap '__catch "${BASH_COMMAND}"' ERR
trap '__cleanup "${tmp_dir}"' EXIT
