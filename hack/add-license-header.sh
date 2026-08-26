#!/usr/bin/env bash

# Copyright 2025 SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
# SPDX-License-Identifier: Apache-2.0


set -e
root_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"
COPYRIGHT="SPDX-FileCopyrightText: Copyright Contributors to the Gardener project"

go tool -modfile=${root_dir}/tools/go.mod addlicense \
  -c "$COPYRIGHT" \
  -l apache \
  -s=only \
  -y "$(date +"%Y")" \
  -ignore "${root_dir}/.git/**" \
  -ignore "${root_dir}/.ci/**" \
  -ignore "${root_dir}/.reuse/**" \
  -ignore "**/*.md" \
  -ignore "**/*.html" \
  -ignore "**/*.yaml" \
  -ignore "**/Dockerfile" \
  ${root_dir}