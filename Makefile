# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

REPO_ROOT                             := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                               := $(shell cat VERSION)
REGISTRY                              := eu.gcr.io/gardener-project/gardener
FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY   := $(REGISTRY)/fluent-bit-to-vali
LOKI_CURATOR_IMAGE_REPOSITORY         := $(REGISTRY)/vali-curator
TELEGRAF_IMAGE_REPOSITORY             := $(REGISTRY)/telegraf-iptables
TUNE2FS_IMAGE_REPOSITORY              := $(REGISTRY)/tune2fs
EVENT_LOGGER_IMAGE_REPOSITORY         := $(REGISTRY)/event-logger
IMAGE_TAG                             := $(VERSION)
EFFECTIVE_VERSION                     := $(VERSION)-$(shell git rev-parse HEAD)
GOARCH                                := amd64

.PHONY: plugin
plugin:
	go build -mod=vendor -buildmode=c-shared -o build/out_vali.so ./cmd/fluent-bit-vali-plugin

.PHONY: curator
curator:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) GO111MODULE=on \
	  go build -mod=vendor -o build/curator ./cmd/vali-curator

.PHONY: event-logger
event-logger:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) GO111MODULE=on \
	  go build -mod=vendor -o build/event-logger ./cmd/event-logger

.PHONY: build
build: plugin

.PHONY: install
install: install-vali-curator install-event-logger

.PHONY: install-vali-curator
install-vali-curator:
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/install.sh ./cmd/vali-curator

.PHONY: install-event-logger
install-event-logger:
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/install.sh ./cmd/event-logger

.PHONY: install-copy
install-copy:
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/install.sh ./cmd/copy

.PHONY: docker-images
docker-images:
	@docker build -t $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY):$(IMAGE_TAG) -t $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY):latest -f Dockerfile --target fluent-bit-plugin .
	@docker build -t $(LOKI_CURATOR_IMAGE_REPOSITORY):$(IMAGE_TAG) -t $(LOKI_CURATOR_IMAGE_REPOSITORY):latest -f Dockerfile --target curator .
	@docker build -t $(TELEGRAF_IMAGE_REPOSITORY):$(IMAGE_TAG) -t $(TELEGRAF_IMAGE_REPOSITORY):latest -f Dockerfile --target telegraf .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(EVENT_LOGGER_IMAGE_REPOSITORY):$(IMAGE_TAG) -t $(EVENT_LOGGER_IMAGE_REPOSITORY):latest -f Dockerfile --target event-logger .
	@docker build -t $(TUNE2FS_IMAGE_REPOSITORY):$(IMAGE_TAG) -t $(TUNE2FS_IMAGE_REPOSITORY):latest -f Dockerfile --target tune2fs .

.PHONY: docker-push
docker-push:
	@if ! docker images $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY):$(IMAGE_TAG)
	@if ! docker images $(LOKI_CURATOR_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(LOKI_CURATOR_IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(LOKI_CURATOR_IMAGE_REPOSITORY):$(IMAGE_TAG)
	@if ! docker images $(TELEGRAF_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(TELEGRAF_IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(TELEGRAF_IMAGE_REPOSITORY):$(IMAGE_TAG)
	@if ! docker images $(EVENT_LOGGER_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(EVENT_LOGGER_IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(EVENT_LOGGER_IMAGE_REPOSITORY):$(IMAGE_TAG)
	@if ! docker images $(TUNE2FS_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(TUNE2FS_IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(TUNE2FS_IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor

.PHONY: check
check:	
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/...

.PHONY: format
format:
	@sh $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg

.PHONY: test
test:
	@sh $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh ./pkg/...

.PHONY: install-requirements
install-requirements:
	@go install -mod=vendor github.com/onsi/ginkgo/ginkgo
	@$(REPO_ROOT)/hack/install-requirements.sh

.PHONY: verify
verify: install-requirements check format test
