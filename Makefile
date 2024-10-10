# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

REPO_ROOT                                  := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                                    := $(shell cat VERSION)
REGISTRY                                   ?= europe-docker.pkg.dev/gardener-project/snapshots/gardener
FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY        := $(REGISTRY)/fluent-bit-to-vali
FLUENT_BIT_VALI_IMAGE_REPOSITORY           := $(REGISTRY)/fluent-bit-vali
VALI_CURATOR_IMAGE_REPOSITORY              := $(REGISTRY)/vali-curator
TELEGRAF_IMAGE_REPOSITORY                  := $(REGISTRY)/telegraf-iptables
TUNE2FS_IMAGE_REPOSITORY                   := $(REGISTRY)/tune2fs
EVENT_LOGGER_IMAGE_REPOSITORY              := $(REGISTRY)/event-logger
IMAGE_TAG                                  := $(VERSION)
EFFECTIVE_VERSION                          := $(VERSION)-$(shell git rev-parse --short HEAD)
SRC_DIRS                                   := $(shell go list -f '{{.Dir}}' $(REPO_ROOT)/...)
LD_FLAGS                                   := $(shell $(REPO_ROOT)/hack/get-build-ld-flags.sh)
BUILD_PLATFORM                             ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
BUILD_ARCH                                 ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
PARALLEL_E2E_TESTS                         := 1

# project folder structure
PKG_DIR                                    := $(REPO_ROOT)/pkg
TOOLS_DIR                                  := $(REPO_ROOT)/tools
GARDENER_DIR                               := $(REPO_ROOT)/gardener
# linter dependencies
GO_LINT                                    := $(TOOLS_DIR)/golangci-lint
GO_LINT_VERSION                            ?= v1.60.3
# test dependencies
GINKGO                                     := $(TOOLS_DIR)/ginkgo
GINKGO_VERSION                             ?= v2.19.0
# kind dependencies
KIND                                       := $(TOOLS_DIR)/kind
KIND_VERSION                               ?= v0.24.0

# kubectl dependencies
KUBECTL                                    := $(TOOLS_DIR)/kubectl
KUBECTL_VERSION                            ?= v1.31.1

# skaffold dependencies
SKAFFOLD                                   := $(TOOLS_DIR)/skaffold
SKAFFOLD_VERSION                           ?= latest
# helm dependencies
HELM                                       := $(TOOLS_DIR)/helm
HELM_VERSION                               ?= v3.12.0

# goimports dependencies
GOIMPORTS                                  := $(TOOLS_DIR)/goimports
GOIMPORTS_VERSION                          ?= v0.22.0
# goimports_reviser dependencies
GOIMPORTS_REVISER                          := $(TOOLS_DIR)/goimports-reviser
GOIMPORTS_REVISER_VERSION                  ?= v3.6.5

$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

export PATH := $(abspath $(TOOLS_DIR)):$(PATH)

.DEFAULT_GOAL := all
all: verify goimports goimports-reviser

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: plugin
plugin: tidy
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		go build -buildmode=c-shared \
		-o $(REPO_ROOT)/build/out_vali.so \
	  	-ldflags="$(LD_FLAGS)" \
		./cmd/fluent-bit-vali-plugin

.PHONY: curator
curator: tidy
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		CGO_ENABLED=0 \
		GO111MODULE=on \
		go build \
		-o $(REPO_ROOT)/build/curator \
		-ldflags="$(LD_FLAGS)" \
		./cmd/vali-curator

.PHONY: event-logger
event-logger: tidy
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		CGO_ENABLED=0 GO111MODULE=on \
		go build \
		-o $(REPO_ROOT)/build/event-logger \
		-ldflags="$(LD_FLAGS)" \
		$(REPO_ROOT)/cmd/event-logger

.PHONY: install
install: install-vali-curator install-event-logger

.PHONY: install-vali-curator
install-vali-curator:
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) \
		./hack/install.sh \
		./cmd/vali-curator

.PHONY: install-event-logger
install-event-logger:
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) \
		./hack/install.sh \
		./cmd/event-logger

.PHONY: install-copy
install-copy:
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) \
		./hack/install.sh \
		./cmd/copy

.PHONY: docker-images
docker-images:
	@BUILD_ARCH=$(BUILD_ARCH) \
		$(REPO_ROOT)/hack/docker-image-build.sh "fluent-bit-plugin" \
		$(FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@BUILD_ARCH=$(BUILD_ARCH) \
    	$(REPO_ROOT)/hack/docker-image-build.sh "fluent-bit-vali" \
    	$(FLUENT_BIT_VALI_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@BUILD_ARCH=$(BUILD_ARCH) \
		$(REPO_ROOT)/hack/docker-image-build.sh "curator" \
		$(VALI_CURATOR_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@BUILD_ARCH=$(BUILD_ARCH) \
		$(REPO_ROOT)/hack/docker-image-build.sh "telegraf" \
		$(TELEGRAF_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@BUILD_ARCH=$(BUILD_ARCH) \
		$(REPO_ROOT)/hack/docker-image-build.sh "event-logger" \
		$(EVENT_LOGGER_IMAGE_REPOSITORY) $(IMAGE_TAG) $(EFFECTIVE_VERSION)

	@BUILD_ARCH=$(BUILD_ARCH) \
		$(REPO_ROOT)/hack/docker-image-build.sh "tune2fs" \
		$(TUNE2FS_IMAGE_REPOSITORY) $(IMAGE_TAG)

.PHONY: docker-push
docker-push:
	@$(REPO_ROOT)/hack/docker-image-push.sh "fluent-bit-plugin" \
	$(FLUENT_BIT_TO_VALI_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@$(REPO_ROOT)/hack/docker-image-push.sh "curator" \
	$(VALI_CURATOR_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@$(REPO_ROOT)/hack/docker-image-push.sh "telegraf" \
	$(TELEGRAF_IMAGE_REPOSITORY) $(IMAGE_TAG)

	@$(REPO_ROOT)/hack/docker-image-push.sh "event-logger" \
	$(EVENT_LOGGER_IMAGE_REPOSITORY) $(IMAGE_TAG) $(EFFECTIVE_VERSION)

	@$(REPO_ROOT)/hack/docker-image-push.sh "tune2fs" \
    $(TUNE2FS_IMAGE_REPOSITORY) $(IMAGE_TAG)

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: tidy
tidy:
	@go mod tidy
	@go mod download

.PHONY: check
check: format $(GO_LINT)
	 @$(GO_LINT) run --config=$(REPO_ROOT)/.golangci.yaml --timeout 10m $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/...
	 @go vet $(REPO_ROOT)/cmd/... $(REPO_ROOT)/pkg/... $(REPO_ROOT)/tests/...

.PHONY: format
format:
	@gofmt -l -w $(REPO_ROOT)/cmd $(REPO_ROOT)/pkg $(REPO_ROOT)/tests

.PHONY: goimports
goimports: $(GOIMPORTS)
	@for dir in $(SRC_DIRS); do \
		$(GOIMPORTS) -w $$dir/; \
	done

.PHONY: goimports-reviser
goimports-reviser: $(GOIMPORTS_REVISER)
	@for dir in $(SRC_DIRS); do \
		GOIMPORTS_REVISER_OPTIONS="-imports-order std,project,general,company" \
		$(GOIMPORTS_REVISER) -recursive $$dir/; \
	done

.PHONY: test
test: $(GINKGO)
	@go test $(REPO_ROOT)/pkg/... --v --ginkgo.v --ginkgo.no-color

.PHONY: install-requirements
install-requirements: $(GO_LINT) $(GINKGO) $(KIND) $(KUBECTL) $(SKAFFOLD) $(GOIMPORTS) $(GOIMPORTS_REVISER)

.PHONY: verify
verify: install-requirements format check test

.PHONY: clean
clean:
	@go clean --modcache --testcache
	@rm -rf $(TOOLS_DIR)
	@( [ -d "$(REPO_ROOT)/build" ] && go clean $(REPO_ROOT)/build ) || true

.PHONY: e2e-tests
e2e-tests: $(KIND)
	go test -v $(REPO_ROOT)/tests/...

#########################################
# skaffold pipeline scenarios           #
#########################################
skaffold-%: export KUBECONFIG = $(REPO_ROOT)/example/kind/kubeconfig

# skaffold and skafold-dev targets require a running kind cluster
# make kind-up

.PHONY: skaffold-run
skaffold-run: $(SKAFFOLD)
	@$(SKAFFOLD) run --kubeconfig=$(KUBECONFIG)

# skaffold-dev target requires that skaffold run has been run
.PHONY: skaffold-dev
skaffold-dev: $(SKAFFOLD)
	@$(SKAFFOLD) dev --kubeconfig=$(KUBECONFIG)


#########################################
# Tools                                 #
#########################################

.PHONY: kind-up
kind-up: $(KIND) $(KUBECTL)
	@$(REPO_ROOT)/hack/kind-up.sh

# fetch linter dependency
$(GO_LINT):
	@GOBIN=$(abspath $(TOOLS_DIR)) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GO_LINT_VERSION)

# fetch ginkgo dependency
$(GINKGO):
	@GOBIN=$(abspath $(TOOLS_DIR)) go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

# fetch kind dependency
$(KIND):
	@curl -L -o $(KIND) https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(BUILD_PLATFORM)-$(BUILD_ARCH)
	@chmod +x $(KIND)

$(KUBECTL):
	@curl -L -o $(KUBECTL) "https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$(BUILD_PLATFORM)/$(BUILD_ARCH)/kubectl"
	@chmod +x $(KUBECTL)

# fetch skaffold dependency
$(SKAFFOLD):
	@curl -L -o $(SKAFFOLD) https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-$(BUILD_PLATFORM)-$(BUILD_ARCH)
	@chmod +x $(SKAFFOLD)

$(GOIMPORTS):
	@GOBIN=$(abspath $(TOOLS_DIR)) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

$(GOIMPORTS_REVISER):
	@GOBIN=$(abspath $(TOOLS_DIR)) go install github.com/incu6us/goimports-reviser/v3@$(GOIMPORTS_REVISER_VERSION)
