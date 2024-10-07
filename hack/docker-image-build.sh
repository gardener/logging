#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

dir="$(dirname "$0")"

set -o nounset
set -o pipefail
set -o errexit

TARGET="${1:-}"
if [ -z $TARGET ]; then
  echo "TARGET is a required parameter" && exit 1
fi

IMAGE_REPOSITORY="${2:-}"
if [ -z $IMAGE_REPOSITORY ]; then
  echo "IMAGE_REPOSITORY is a required parameter" && exit 1
fi

IMAGE_TAG="${3:-}"
if [ -z $IMAGE_TAG ]; then
  echo "IMAGE_TAG is a required parameter" && exit 1
fi

EFFECTIVE_VERSION="${4:-}"

echo "docker build: ${TARGET} for linux/${BUILD_ARCH}"
docker build \
  --build-arg EFFECTIVE_VERSION="${EFFECTIVE_VERSION}" \
  --build-arg LD_FLAGS="$($dir/get-build-ld-flags.sh)" \
  --tag "${IMAGE_REPOSITORY}:latest" \
	--tag "${IMAGE_REPOSITORY}:${IMAGE_TAG}" \
  --platform "linux/${BUILD_ARCH}" \
	-f Dockerfile --target ${TARGET} $dir/..
