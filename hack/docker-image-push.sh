#!/usr/bin/env bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

root_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"

set -o nounset #catch and prevent errors caused by the use of unset variables.
set -o pipefail #exit with the exit code of the first error
set -o errexit #exits immediately if any command in a script exits with a non-zero status

source "${root_dir}/hack/.includes.sh"

echo "> Docker image push"

TARGET="${1:-}"
if [ -z $TARGET ]; then
  echo "TARGET is a required parameter"  && exit 1
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

# Check if the image with 'latest' tag exists
if __image_exists "${IMAGE_REPOSITORY}:latest"; then
    echo "Docker image '${IMAGE_REPOSITORY}:latest' found."
else
    echo "Error: Docker image '${IMAGE_REPOSITORY}:latest' not found."
    exit 1
fi

# Check if the image with specific tag exists
if __image_exists "${IMAGE_REPOSITORY}:${IMAGE_TAG}"; then
    echo "Docker image '${IMAGE_REPOSITORY}:${IMAGE_TAG}' found."
else
    echo "Error: Docker image '${IMAGE_REPOSITORY}:${IMAGE_TAG}' not found."
    exit 1
fi

echo "Pushing Docker image ..."
docker push "${IMAGE_REPOSITORY}:latest"
docker push "${IMAGE_REPOSITORY}:${IMAGE_TAG}"
