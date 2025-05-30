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
EFFECTIVE_VERSION                          := $(VERSION)-$(shell git rev-parse --short HEAD)
SRC_DIRS                                   := $(shell go list -f '{{.Dir}}' $(REPO_ROOT)/...)
LD_FLAGS                                   := -s -w $(shell $(REPO_ROOT)/hack/get-build-ld-flags.sh)
BUILD_PLATFORM                             ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
BUILD_ARCH                                 ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

GCI_OPT                                    ?= -s standard -s default -s "prefix($(shell go list -m))" --skip-generated

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif
IMAGE_TAG                                  := $(EFFECTIVE_VERSION)

# project folder structure
TOOLS_DIR                                  := $(REPO_ROOT)/tools
include hack/tools.mk
export PATH := $(abspath $(TOOLS_DIR)):$(PATH)

.DEFAULT_GOAL := all
all: verify plugin curator event-logger

#################################################################
# Build targets                                                 #
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

.PHONY: copy
copy: tidy
	@echo "building $@ for $(BUILD_PLATFORM)/$(BUILD_ARCH)"
	@GOOS=$(BUILD_PLATFORM) \
		GOARCH=$(BUILD_ARCH) \
		CGO_ENABLED=0 GO111MODULE=on \
		go build \
		-o $(REPO_ROOT)/build/copy \
		-ldflags="$(LD_FLAGS)" \
		$(REPO_ROOT)/cmd/copy

#################################################################
# Container imges build targets                                 #
#################################################################
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

#################################################################
# Code check targets                                            #
#################################################################

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: check
check: tidy fmt gci lint

.PHONY: fmt
fmt: tidy
	@echo "Running fmt..."
	@go tool golangci-lint fmt \
	 	--config=$(REPO_ROOT)/.golangci.yaml \
		$(SRC_DIRS)

.PHONY: gci
gci: tidy
	@echo "Running gci..."
	@go tool gci write $(GCI_OPT) $(SRC_DIRS)

.PHONY: lint
check: tidy
	@echo "Running lint..."
	 @go tool golangci-lint run \
	 	--config=$(REPO_ROOT)/.golangci.yaml \
		$(SRC_DIRS)

.PHONY: test
test: tidy
	@go tool gotestsum $(REPO_ROOT)/pkg/... --v --ginkgo.v --ginkgo.no-color
	@go tool gotestsum $(REPO_ROOT)/tests/valiplugin

.PHONY: e2e-tests
e2e-tests: tidy
	@KIND_PATH=$(shell go tool -n kind) go tool gotestsum $(REPO_ROOT)/tests/e2e

.PHONY: verify
verify: check test

.PHONY: sast
sast: $(GOSEC)
	@$(REPO_ROOT)/hack/sast.sh

.PHONY: sast-report
sast-report: $(GOSEC)
	@$(REPO_ROOT)/hack/sast.sh --gosec-report true

.PHONY: add-license-headers
add-license-headers: tidy
	@$(REPO_ROOT)/hack/add-license-header.sh


.PHONY: clean
clean:
	@rm -rf $(REPO_ROOT)/build

#########################################
# Tools                                 #
#########################################
.PHONY: kind-up
kind-up: tidy $(KUBECTL)
	@$(REPO_ROOT)/hack/kind-up.sh

#########################################
# skaffold pipeline scenarios           #
#########################################
skaffold-%: export KUBECONFIG = $(REPO_ROOT)/example/kind/kubeconfig

.PHONY: skaffold-run
skaffold-run: $(SKAFFOLD)
	@$(SKAFFOLD) run --kubeconfig=$(KUBECONFIG)

# skaffold-dev target requires that skaffold run has been run
.PHONY: skaffold-dev
skaffold-dev: $(SKAFFOLD)
	@$(SKAFFOLD) dev --kubeconfig=$(KUBECONFIG)
