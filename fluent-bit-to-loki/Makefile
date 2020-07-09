REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION                               := $(shell cat VERSION)
REGISTRY                              := eu.gcr.io/gardener-project/gardener
FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY   := $(REGISTRY)/fluent-bit-to-loki
FLUENT_BIT_TO_LOKI_IMAGE_TAG          := $(VERSION)

.PHONY: plugin
plugin:
	go build -buildmode=c-shared -o build/out_loki.so ./cmd

.PHONY: docker-images
docker-images:
	@docker build -t $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY):$(FLUENT_BIT_TO_LOKI_IMAGE_TAG) -t $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY):latest -f Dockerfile --target fluent-bit .

.PHONY: docker-push
docker-push:
	@if ! docker images $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(FLUENT_BIT_TO_LOKI_IMAGE_TAG); then echo "$(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY) version $(FLUENT_BIT_TO_LOKI_IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(FLUENT_BIT_TO_LOKI_IMAGE_REPOSITORY):$(FLUENT_BIT_TO_LOKI_IMAGE_TAG)

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy

.PHONY: install-requirments
check:
	@$(REPO_ROOT)/hack/install-requirments.sh

.PHONY: check
check:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/...

.PHONY: format
format:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg

.PHONY: test
test:
	@sh $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh -r ./pkg/...

.PHONY: install-requirements
install-requirements:
	@$(REPO_ROOT)/hack/install-requirements.sh

.PHONY: verify
verify: install-requirments check format test
