#!/usr/bin/env bash

repo_root="$(readlink -f $(dirname ${0})/..)"
gardener=$(grep "github.com/gardener/gardener" $repo_root/go.mod |cut -d " " -f2)


function __catch() {
  local cmd="${1:-}"
  echo
  echo "errexit $cmd on line $(caller)" >&2
}
trap '__catch "${BASH_COMMAND}"' ERR

function __check_executables {  
  # required execs
  execs=(make docker)
  for ex in ${execs[@]}; do
    if ! command -v "$ex" &> /dev/null ; then echo "$ex is required"; return 1; fi  
  done
  return 0
}

__check_executables

# fetch yq in tools if not present
if [[ ! -f "$repo_root/tools/yq" ]]; then
  make -C $repo_root "$repo_root/tools/yq" 
fi

# fetch kind in tools if not present
if [[ ! -f "$repo_root/tools/kind" ]]; then
  make -C $repo_root "$repo_root/tools/kind" 
fi