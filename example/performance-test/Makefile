# Plugin Performance Test Makefile
# This Makefile provides targets to run performance tests for the logging plugin.
DIR                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
PLATFORM              ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH                  ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# Default values - suitable for test environment verification
# These can be overridden via environment variables for actual performance testing.
CLUSTERS              ?= 20 # Number of clusters to create
JOBS                  ?= 10 # Number of parallel jobs per cluster
LOGS                  ?= 1000 # Number of log lines per job
LOGS_DELAY            ?= 25s # Delay between log messages

SCENARIO              ?= shoot # Test scenario: "shoot" or "seed"

VLOGSCLI_BIN            := $(DIR)/bin/vlogscli
VLOGSCLI_BIN_VERSION    := v1.39.0
VLOGSCLI_URL 		    := "https://github.com/VictoriaMetrics/VictoriaLogs/releases/download/$(VLOGSCLI_BIN_VERSION)/vlutils-$(PLATFORM)-$(ARCH)-$(VLOGSCLI_BIN_VERSION).tar.gz"

# Default target
.PHONY: help vlogscli setup run fetch down clean test check check-kubeconfig
help:
	@echo "Available targets:"
	@echo "  setup  - Install fluent-bit chart and apply cluster CRD"
	@echo "  run    - Create test clusters and start log generation jobs"
	@echo "  fetch  - Query Vali to validate log ingestion"
	@echo "  check  - Show current number of logs in Vali"
	@echo "  down   - Clean up test clusters and log generation jobs"
	@echo "  clean  - Remove all test components"
	@echo "  vlogsqcli - Download and install vlogsqcli tool"
	@echo ""
	@echo "Environment variables:"
	@echo "  CLUSTERS    - Number of clusters to create (default: $(CLUSTERS))"
	@echo "  JOBS        - Number of parallel jobs per cluster (default: $(JOBS))"
	@echo "  LOGS        - Number of logs per job (default: $(LOGS))"
	@echo "  LOGS_DELAY  - Delay between log messages (default: $(LOGS_DELAY))"

# Download and install logcli tool using a dedicated Go module

vlogscli: $(VLOGSCLI_BIN)
    @echo "vlogscli present: $(VLOGSCLI_BIN)"

$(VLOGSCLI_BIN):
	@echo "Downloading vlogscli from victorialogs"
	@mkdir -p $(DIR)/bin
	@TMP_DIR=$$(mktemp -d) && \
	    cd $$TMP_DIR && \
		curl -L -O -s $(VLOGSCLI_URL) && \
	    tar xzf vlutils-$(PLATFORM)-$(ARCH)-$(VLOGSCLI_BIN_VERSION).tar.gz && \
		mv ./vlogscli-prod $(DIR)/bin/vlogscli && \
		chmod +x $(DIR)/bin/vlogscli
	@rm -rf $$TMP_DIR
	@$(DIR)/bin/vlogscli -version

# Install fluent-bit chart and apply cluster CRD
setup: vlogscli check-kubeconfig
	@echo "Setting up logging performance test environment..."
	@$(DIR)/setup.sh

# Create test clusters and start log generation
run: check-kubeconfig setup
	@echo "Starting performance test with $(CLUSTERS) clusters, $(JOBS) jobs per cluster, $(LOGS) logs per job..."
	@export CLUSTERS=$(CLUSTERS) JOBS=$(JOBS) LOGS=$(LOGS) LOGS_DELAY=$(LOGS_DELAY) && $(DIR)/run.sh $(SCENARIO)

# Check Current logs in Vali
check: check-kubeconfig
	@echo "Fetching total logs ingested so far..."
	@$(DIR)/check.sh

# Query Vali to validate log ingestion
fetch: check-kubeconfig
	@echo "Fetching and validating log ingestion results..."
	@export CLUSTERS=$(CLUSTERS) JOBS=$(JOBS) LOGS=$(LOGS) && $(DIR)/fetch.sh

# Clean up all test resources
down: check-kubeconfig
	@echo "Removing performance test workloads resources..."
	@export CLUSTERS=$(CLUSTERS) && $(DIR)/down.sh $(SCENARIO)

# Alias for down
clean: check-kubeconfig down
	@echo "Cleaning up performance test components..."
	@$(DIR)/clean.sh

# Run complete test cycle
test: setup run
	@echo "Waiting for logs to be ingested..."
	@$(MAKE) fetch

# Show target cluster
check-kubeconfig:
	@if [ -z "$$KUBECONFIG" ]; then \
		echo "KUBECONFIG is not set"; \
		exit 1; \
	fi
	@echo "Target cluster is $$(kubectl config current-context)"
