#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

dir="$(dirname "$0")"

set -o nounset #catch and prevent errors caused by the use of unset variables.
set -o pipefail #exit with the exit code of the first error
set -o errexit #exits immediately if any command in a script exits with a non-zero status

source "$dir/.includes.sh"

echo "> Docker build"

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

pushd $dir/..

docker build \
  --build-arg EFFECTIVE_VERSION="${EFFECTIVE_VERSION}" \
  --tag "${IMAGE_REPOSITORY}:latest" \
	--tag "${IMAGE_REPOSITORY}:${IMAGE_TAG}" \
	-f Dockerfile --target ${TARGET} .

popd      