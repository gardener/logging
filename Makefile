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

REPO_ROOT                                  := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                    := $(shell cat VERSION)
REGISTRY                                   ?= eu.gcr.io/gardener-project/gardener
FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY        := $(REGISTRY)/fluent-bit-to-vali
VALI_CURATOR_IMAGE_REPOSITORY              := $(REGISTRY)/vali-curator
TELEGRAF_IMAGE_REPOSITORY                  := $(REGISTRY)/telegraf-iptables
TUNE2FS_IMAGE_REPOSITORY                   := $(REGISTRY)/tune2fs
EVENT_LOGGER_IMAGE_REPOSITORY              := $(REGISTRY)/event-logger
IMAGE_TAG                                  := $(VERSION)
EFFECTIVE_VERSION                          := $(VERSION)-$(shell git rev-parse HEAD)
PARALLEL_E2E_TESTS                         := 1
DOCKER_BUILD_PLATFORM                      ?= linux/amd64,linux/arm64

BUILD_PLATFORM                             :=$(shell uname -s | tr '[:upper:]' '[:lower:]')
BUILD_ARCH                                 :=$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# project folder structure
PKG_DIR                                    := $(REPO_ROOT)/pkg
TOOLS_DIR                                  := $(REPO_ROOT)/tools

# linter dependencies
GO_LINT                                    := $(TOOLS_DIR)/golangci-lint
GO_LINT_VERSION                            ?= v1.51.2

# test dependencies
GINKGO                                     := $(TOOLS_DIR)/ginkgo
GINKGO_VERSION                             ?= v1.16.2

# yq dependencies
YQ                                         := $(TOOLS_DIR)/yq
YQ_VERSION                                 ?= v4.31.2

# kind dependencies
KIND                                       := $(TOOLS_DIR)/kind
KIND_VERSION                               ?= v0.18.0

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: plugin
plugin:
	go build -mod=vendor -buildmode=c-shared -o build/out_vali.so ./cmd/fluent-bit-vali-plugin

.PHONY: curator
curator:
	CGO_ENABLED=0 GO111MODULE=on \
	  go build -mod=vendor -o build/curator ./cmd/vali-curator

.PHONY: event-logger
event-logger:
	CGO_ENABLED=0 GO111MODULE=on \
	  go build -mod=vendor -o $(REPO_ROOT)/build/event-logger $(REPO_ROOT)/cmd/event-logger

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
	@$(REPO_ROOT)/hack/docker-image-build.sh "fluent-bit-plugin" \
	$(FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@$(REPO_ROOT)/hack/docker-image-build.sh "curator" \
	$(VALI_CURATOR_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@$(REPO_ROOT)/hack/docker-image-build.sh "telegraf" \
	$(TELEGRAF_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@$(REPO_ROOT)/hack/docker-image-build.sh "event-logger" \
	$(EVENT_LOGGER_IMAGE_REPOSITORY) $(IMAGE_TAG) $(EFFECTIVE_VERSION)

	@$(REPO_ROOT)/hack/docker-image-build.sh "tune2fs" \
	$(TUNE2FS_IMAGE_REPOSITORY) $(IMAGE_TAG)

.PHONY: docker-push
docker-push:
	@$(REPO_ROOT)/hack/docker-image-push.sh "fluent-bit-plugin" \
	"$(DOCKER_BUILD_PLATFORM)" $(FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY) $(IMAGE_TAG)
	
	@$(REPO_ROOT)/hack/docker-image-push.sh "curator" \
	"$(DOCKER_BUILD_PLATFORM)" $(VALI_CURATOR_IMAGE_REPOSITORY) $(IMAGE_TAG)
	
	@$(REPO_ROOT)/hack/docker-image-push.sh "telegraf" \
	"$(DOCKER_BUILD_PLATFORM)" $(TELEGRAF_IMAGE_REPOSITORY) $(IMAGE_TAG)
	
	@$(REPO_ROOT)/hack/docker-image-push.sh "event-logger" \
	"$(DOCKER_BUILD_PLATFORM)" $(EVENT_LOGGER_IMAGE_REPOSITORY) $(IMAGE_TAG) $(EFFECTIVE_VERSION)
	
	@$(REPO_ROOT)/hack/docker-image-push.sh "tune2fs" \
    "$(DOCKER_BUILD_PLATFORM)" $(TUNE2FS_IMAGE_REPOSITORY) $(IMAGE_TAG)

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor

.PHONY: check
check: format $(GO_LINT)
	 $(GO_LINT) run --config=$(REPO_ROOT)/.golangci.yaml --timeout 10m $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/...
	 go vet -mod=vendor $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/...

.PHONY: format
format:
	gofmt -l -w $(REPO_ROOT)/cmd $(REPO_ROOT)/pkg

.PHONY: test
test: $(GINKGO)
	GO111MODULE=on $(GINKGO) -mod=vendor ./pkg/...

.PHONY: install-requirements
install-requirements: $(GO_LINT) $(GINKGO)

.PHONY: verify
verify: install-requirements check format test

.PHONY: clean
clean:
	@go clean --modcache --testcache
	@rm -rf $(TOOLS_DIR)
	@rm -rf "$(REPO_ROOT)/gardener"
	@( [ -d "$(REPO_ROOT)/build" ] && go clean $(REPO_ROOT)/build ) || true

.PHONY: test-e2e-local
test-e2e-local: $(KIND) $(YQ) $(GINKGO)
	@$(REPO_ROOT)/hack/test-e2e-local.sh --procs=$(PARALLEL_E2E_TESTS) --label-filter "Shoot && simple" ./tests/e2e/...

#########################################
# Tools                                 #
#########################################

# fetch linter dependency
$(GO_LINT):
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GO_LINT_VERSION)

# fetch ginkgo dependency
$(GINKGO):
	@GOBIN=$(TOOLS_DIR) go install github.com/onsi/ginkgo/ginkgo@$(GINKGO_VERSION)

# fetch yq dependency
$(YQ):
	curl -L -o $(YQ) https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$(BUILD_PLATFORM)_$(BUILD_ARCH)
	chmod +x $(YQ)

# fetch kind dependency
$(KIND):
	curl -L -o $(KIND) https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(BUILD_PLATFORM)-$(BUILD_ARCH)
	chmod +x $(KIND)
