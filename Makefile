# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT                                  := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                    := $(shell cat VERSION)
REGISTRY                                   ?= europe-docker.pkg.dev/gardener-project/snapshots/gardener
FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY        := $(REGISTRY)/fluent-bit-to-vali
VALI_CURATOR_IMAGE_REPOSITORY              := $(REGISTRY)/vali-curator
TELEGRAF_IMAGE_REPOSITORY                  := $(REGISTRY)/telegraf-iptables
TUNE2FS_IMAGE_REPOSITORY                   := $(REGISTRY)/tune2fs
EVENT_LOGGER_IMAGE_REPOSITORY              := $(REGISTRY)/event-logger
IMAGE_TAG                                  := $(VERSION)
EFFECTIVE_VERSION                          := $(VERSION)-$(shell git rev-parse HEAD)
PARALLEL_E2E_TESTS                         := 1
DOCKER_BUILD_PLATFORM                      ?= linux/amd64,linux/arm64

LD_FLAGS                                   :=$(shell $(REPO_ROOT)/hack/get-build-ld-flags.sh)
BUILD_PLATFORM                             :=$(shell uname -s | tr '[:upper:]' '[:lower:]')
BUILD_ARCH                                 :=$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# project folder structure
PKG_DIR                                    := $(REPO_ROOT)/pkg
TOOLS_DIR                                  := $(REPO_ROOT)/tools
GARDENER_DIR                               := $(REPO_ROOT)/gardener

# linter dependencies
GO_LINT                                    := $(TOOLS_DIR)/golangci-lint
GO_LINT_VERSION                            ?= v1.59.1

# test dependencies
GINKGO                                     := $(TOOLS_DIR)/ginkgo
GINKGO_VERSION                             ?= v2.19.0

# yq dependencies
YQ                                         := $(TOOLS_DIR)/yq
YQ_VERSION                                 ?= v4.31.2

# kind dependencies
KIND                                       := $(TOOLS_DIR)/kind
KIND_VERSION                               ?= v0.18.0

# skaffold dependencies
SKAFFOLD                                   := $(TOOLS_DIR)/skaffold
SKAFFOLD_VERSION                           ?= latest

# helm dependencies
HELM                                       := $(TOOLS_DIR)/helm
HELM_VERSION                               ?= v3.12.0

export PATH := $(abspath $(TOOLS_DIR)):$(PATH)
#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: plugin
plugin: tidy
	@go build -buildmode=c-shared -o $(REPO_ROOT)/build/out_vali.so \
	  -ldflags="$(LD_FLAGS)" ./cmd/fluent-bit-vali-plugin

.PHONY: curator
curator: tidy
	@CGO_ENABLED=0 GO111MODULE=on go build -o $(REPO_ROOT)/build/curator \
	  -ldflags="$(LD_FLAGS)" ./cmd/vali-curator

.PHONY: event-logger
event-logger: tidy
	@CGO_ENABLED=0 GO111MODULE=on go build -o $(REPO_ROOT)/build/event-logger \
	  -ldflags="$(LD_FLAGS)" $(REPO_ROOT)/cmd/event-logger

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

.PHONY: tidy
tidy:
	go mod tidy
	go mod download

.PHONY: check
check: format $(GO_LINT)
	 $(GO_LINT) run --config=$(REPO_ROOT)/.golangci.yaml --timeout 10m $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/...
	 go vet $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/...

.PHONY: format
format:
	gofmt -l -w $(REPO_ROOT)/cmd $(REPO_ROOT)/pkg

.PHONY: test
test: $(GINKGO)
	$(GINKGO) ./pkg/...

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
test-e2e-local: $(KIND) $(YQ) $(GINKGO) $(GARDENER_DIR)
	@$(REPO_ROOT)/hack/test-e2e-local.sh --procs=$(PARALLEL_E2E_TESTS) --label-filter "Shoot && simple" ./tests/e2e/...

#########################################
# skaffold pipeline scenarios           #
#########################################
skaffold-%: export KUBECONFIG = $(REPO_ROOT)/gardener/example/gardener-local/kind/local/kubeconfig

# skaffold and skafold-dev targets require a running kind cluster initated from a fetched gardener repo (hack/fetch-gardener.sh)
# make -C gardener kind-up

.PHONY: skaffold-run
skaffold-run: $(SKAFFOLD) $(HELM) $(YQ) $(GARDENER_DIR)
	@$(SKAFFOLD) run --kubeconfig=$(KUBECONFIG)

# skaffold-dev target requires that skaffold run has been run
.PHONY: skaffold-dev
skaffold-dev: $(SKAFFOLD) $(HELM) $(YQ) $(GARDENER_DIR)
	@$(SKAFFOLD) dev --kubeconfig=$(KUBECONFIG) -m gardenlet

# fetch gardener repo into the project dir
$(GARDENER_DIR):
	$(REPO_ROOT)/hack/fetch-gardener.sh

#########################################
# Tools                                 #
#########################################

# fetch linter dependency
$(GO_LINT):
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GO_LINT_VERSION)

# fetch ginkgo dependency
$(GINKGO):
	@GOBIN=$(TOOLS_DIR) go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

# fetch yq dependency
$(YQ):
	mkdir -p $(TOOLS_DIR)
	curl -L -o $(YQ) https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$(BUILD_PLATFORM)_$(BUILD_ARCH)
	chmod +x $(YQ)

# fetch kind dependency
$(KIND):
	mkdir -p $(TOOLS_DIR)
	curl -L -o $(KIND) https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(BUILD_PLATFORM)-$(BUILD_ARCH)
	chmod +x $(KIND)

# fetch skaffold dependency
$(SKAFFOLD):
	mkdir -p $(TOOLS_DIR)
	curl -L -o $(SKAFFOLD) https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-$(BUILD_PLATFORM)-$(BUILD_ARCH)
	chmod +x $(SKAFFOLD)

# fetch helm dependency
$(HELM):
	mkdir -p $(TOOLS_DIR)
	curl -sSfL https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | HELM_INSTALL_DIR=$(TOOLS_DIR) USE_SUDO=false bash -s -- --version $(HELM_VERSION)
