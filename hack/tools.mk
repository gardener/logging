# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# kubectl dependency
KUBECTL                                    := $(TOOLS_DIR)/kubectl
KUBECTL_VERSION                            ?= v1.31.1

# skaffold dependency
SKAFFOLD                                   := $(TOOLS_DIR)/skaffold
SKAFFOLD_VERSION                           ?= v2.13.2

# goimports dependencies
GOIMPORTS                                  := $(TOOLS_DIR)/goimports
GOIMPORTS_VERSION                          ?= $(call version_gomod,golang.org/x/tools)

# goimports_reviser dependencies
GOIMPORTS_REVISER                          := $(TOOLS_DIR)/goimports-reviser
GOIMPORTS_REVISER_VERSION                  ?= v3.6.5

MOCKGEN                                    := $(TOOLS_DIR)/mockgen
MOCKGEN_VERSION                            ?= $(call version_gomod,go.uber.org/mock)

GO_ADD_LICENSE                             := $(TOOLS_DIR)/addlicense
GO_ADD_LICENSE_VERSION                     ?= $(call version_gomod,github.com/google/addlicense)

# gosec
GOSEC     	                           := $(TOOLS_DIR)/gosec
GOSEC_VERSION		                   ?= v2.21.4

# Use this "function" to add the version file as a prerequisite for the tool target: e.g.
tool_version_file = $(TOOLS_DIR)/.version_$(subst $(TOOLS_DIR)/,,$(1))_$(2)
# Use this function to get the version of a go module from go.mod
version_gomod = $(shell go list -mod=mod -f '{{ .Version }}' -m $(1))

$(TOOLS_DIR)/.version_%:
	@version_file=$@; rm -f $${version_file%_*}*
	@mkdir -p $(TOOLS_DIR)
	@touch $@

.PHONY: clean-tools
clean-tools:
	@rm -rf $(TOOLS_DIR)/*

.PHONY: create-tools
create-tools: $(MOCKGEN) $(GOIMPORTS) $(GOIMPORTS_REVISER) $(KUBECTL) $(SKAFFOLD)

$(MOCKGEN): $(call tool_version_file,$(MOCKGEN),$(MOCKGEN_VERSION))
	@echo "install target: $@"
	@go build -o $(MOCKGEN) go.uber.org/mock/mockgen

$(GOIMPORTS): $(call tool_version_file,$(GOIMPORTS),$(GOIMPORTS_VERSION))
	@echo "install target: $@"
	@go build -o $(GOIMPORTS) golang.org/x/tools/cmd/goimports

$(GOIMPORTS_REVISER): $(call tool_version_file,$(GOIMPORTS_REVISER),$(GOIMPORTS_REVISER_VERSION))
	@echo "install target: $@"
	@go build -o $(GOIMPORTS_REVISER) github.com/incu6us/goimports-reviser/v3

$(GOSEC): $(call tool_version_file,$(GOSEC),$(GOSEC_VERSION))
	@echo "install target: $@"
	@GOBIN=$(abspath $(TOOLS_DIR)) go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

$(KUBECTL): $(call tool_version_file,$(KUBECTL),$(KUBECTL_VERSION))
	@echo "install target: $@"
	@curl -sSL -o $(KUBECTL) "https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$(BUILD_PLATFORM)/$(BUILD_ARCH)/kubectl"
	@chmod +x $(KUBECTL)

$(SKAFFOLD): $(call tool_version_file,$(SKAFFOLD),$(SKAFFOLD_VERSION))
	@echo "install target: $@"
	@curl -sSL -o $(SKAFFOLD) "https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-$(BUILD_PLATFORM)-$(BUILD_ARCH)"
	@chmod +x $(SKAFFOLD)

$(GO_ADD_LICENSE):  $(call tool_version_file,$(GO_ADD_LICENSE),$(GO_ADD_LICENSE_VERSION))
	@go build -o $(GO_ADD_LICENSE) github.com/google/addlicense
