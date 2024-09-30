#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

dir="$(dirname "$0")"

set -o nounset
set -o pipefail
set -o errexit

echo "install: $@"

LD_FLAGS="${LD_FLAGS:-$($(dirname $0)/get-build-ld-flags.sh)}"
CGO_ENABLED=0 GOOS=$(go env BUILD_PLATFORM ) GOARCH=$(go env BUILD_ARCH) GO111MODULE=on \
  go install -ldflags "$LD_FLAGS" \
  $@
