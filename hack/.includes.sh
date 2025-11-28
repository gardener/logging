#!/usr/bin/env bash
# Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0


set -o errexit

root_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"
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

function __image_exists {
    local image="${1:-}"
    docker image inspect "$image" > /dev/null 2>&1
}
