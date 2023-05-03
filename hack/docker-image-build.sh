#!/usr/bin/env bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
dir=$(dirname $0)

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
  --tag "${IMAGE_REPOSITORY}:${IMAGE_TAG}-latest" \
	--tag "${IMAGE_REPOSITORY}:${IMAGE_TAG}" \
	-f Dockerfile --target ${TARGET} .

popd      