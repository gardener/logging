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
CURATOR_ES_IMAGE_REPOSITORY           := $(REGISTRY)/curator-es
CURATOR_ES_IMAGE_TAG                  := $(VERSION)
FLUENTD_ES_IMAGE_REPOSITORY           := $(REGISTRY)/fluentd-es
FLUENTD_ES_IMAGE_TAG                  := $(VERSION)

.PHONY: docker-images
docker-images: curator-es-docker-image fluentd-es-docker-image fluent-bit-to-loki-image

.PHONY: curator-es-docker-image
curator-es-docker-image:
	@docker build -t $(CURATOR_ES_IMAGE_REPOSITORY):$(CURATOR_ES_IMAGE_TAG) -f curator-es/Dockerfile --rm .

.PHONY: fluentd-es-docker-image
fluentd-es-docker-image:
	@docker build -t $(FLUENTD_ES_IMAGE_REPOSITORY):$(FLUENTD_ES_IMAGE_TAG) -f fluentd-es/Dockerfile --rm .

.PHONY: fluent-bit-to-loki-image
fluent-bit-to-loki-image:
	@cd fluent-bit-to-loki && $(MAKE) make docker-images

.PHONY: release
release: docker-images docker-login docker-push

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-push
docker-push:
	@if ! docker images $(CURATOR_ES_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(CURATOR_ES_IMAGE_TAG); then echo "$(CURATOR_ES_IMAGE_REPOSITORY) version $(CURATOR_ES_IMAGE_TAG) is not yet built. Please run 'make curator-es-docker-image'"; false; fi
	@if ! docker images $(FLUENTD_ES_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(FLUENTD_ES_IMAGE_TAG); then echo "$(FLUENTD_ES_IMAGE_REPOSITORY) version $(FLUENTD_ES_IMAGE_TAG) is not yet built. Please run 'make fluentd-es-docker-image'"; false; fi
	@gcloud docker -- push $(CURATOR_ES_IMAGE_REPOSITORY):$(CURATOR_ES_IMAGE_TAG)
	@gcloud docker -- push $(FLUENTD_ES_IMAGE_REPOSITORY):$(FLUENTD_ES_IMAGE_TAG)
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
