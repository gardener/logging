# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

VERSION                               := $(shell cat VERSION)
REGISTRY                              := eu.gcr.io/gardener-project/gardener

.PHONY: docker-images
docker-images: fluent-bit-to-loki-image

.PHONY: fluent-bit-to-loki-image
fluent-bit-to-loki-image:
	@cd fluent-bit-to-loki && $(MAKE) docker-images

.PHONY: release
release: docker-images docker-login docker-push

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-push
docker-push:
	cd fluent-bit-to-loki && $(MAKE) docker-push

.PHONY: check
check:
	@.ci/check
	@.ci/verify

.PHONY: format
format:
	@cd fluent-bit-to-loki && $(MAKE) format

.PHONY: test
test:
	@cd fluent-bit-to-loki && $(MAKE) test

.PHONY: verify
verify:
	@cd fluent-bit-to-loki && $(MAKE) verify

.PHONY: revendor
revendor:
	@cd fluent-bit-to-loki && $(MAKE) revendor

.PHONY: install-requirements
install-requirements:
	@cd fluent-bit-to-loki && $(MAKE) install-requirements
