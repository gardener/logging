# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# kubectl dependency
KUBECTL                                    := $(TOOLS_DIR)/kubectl
KUBECTL_VERSION                            ?= v1.32.0

# skaffold dependency
SKAFFOLD                                   := $(TOOLS_DIR)/skaffold
SKAFFOLD_VERSION                           ?= v2.14.1

# gosec
GOSEC     	                               := $(TOOLS_DIR)/gosec
GOSEC_VERSION		                       ?= v2.21.4

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
create-tools:$(GOSEC) $(KUBECTL) $(SKAFFOLD)

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
