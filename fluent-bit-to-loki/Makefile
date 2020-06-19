
REGISTRY                           := eu.gcr.io/gardener-project/gardener
PLUGIN_REPOSITORY         		   := $(REGISTRY)/fluent-bit-to-loki
IMAGE_TAG                          := $(shell cat VERSION)
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: plugin
plugin:
	go build -buildmode=c-shared -o build/out_loki.so ./cmd

.PHONY: docker-images
docker-images:
	@docker build -t $(PLUGIN_REPOSITORY):$(IMAGE_TAG) -t $(PLUGIN_REPOSITORY):latest -f Dockerfile --target fluent-bit .

.PHONY: docker-push
docker-push:
	@if ! docker images $(PLUGIN_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(PLUGIN_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(PLUGIN_REPOSITORY):$(IMAGE_TAG)
	@if [[ "$(PUSH_LATEST_TAG)" == "true" ]]; then gcloud docker -- push $(PLUGIN_REPOSITORY):latest; fi

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy

.PHONY: check
check:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/...

.PHONY: format
format:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg

.PHONY: test
test:
	@sh $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh -r ./pkg/...

.PHONY: verify
verify: check format test