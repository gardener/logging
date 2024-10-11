#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e
dir=$(dirname $0)
repo_root=$(realpath $dir/..)

echo "> Adding Apache License header to all go files where it is not present"

# addlicence with a license file (parameter -f) expects no comments in the file.
# LICENSE_BOILERPLATE.txt is however also used also when generating go code.
# Therefore we remove '//' from LICENSE_BOILERPLATE.txt here before passing it to addlicense.

temp_file=$(mktemp)
trap "rm -f $temp_file" EXIT
sed 's|^// *||' $repo_root/hack/LICENSE_BOILERPLATE.txt > $temp_file

$repo_root/tools/addlicense \
  -f $temp_file \
  -ignore "$repo_root/.idea/**" \
  -ignore "$repo_root/.vscode/**" \
  -ignore "$repo_root/.scripts/**" \
  -ignore "$repo_root/tools/**" \
  -ignore "$repo_root/cmd/fluent-bit-vali-plugin/out_vali.h" \
  -ignore "$repo_root/Dockerfile" \
  -ignore "**/*.md" \
  -ignore "**/*.yaml" \
  -ignore "**/*.conf" \
  $repo_root
