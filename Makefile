# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
include make/quality.mk

GOPATH ?= $(shell go env GOPATH)

CATALOG_REPO=https://github.com/oracle-cne/catalog.git
MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
OCNE_DIR:=github.com/oracle-cne$(shell echo ${MAKEFILE_DIR} | sed 's/.*github.com//')
CLONE_DIR:=${MAKEFILE_DIR}/temp-clone-dir
BUILD_DIR:=build
OUT_DIR:=out
PLATFORM_OUT_DIR:=$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)
PLATFORM_INSTRUMENTED_OUT_DIR:=$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented 
CHART_BUILD_DIR:=$(BUILD_DIR)/catalog
CHART_BUILD_OUT_DIR:=$(CHART_BUILD_DIR)/repo
CHART_GIT_DIR:=build/charts

CHART_EMBED:=pkg/catalog/embedded/charts

TEST_PATTERN:=.*
TEST_FILTERS:=
TEST_DIR:=$(OUT_DIR)/tests
BATS_RESULT_DIR:=$(MAKEFILE_DIR)/$(TEST_DIR)/results
GOCOVERDIR:=$(MAKEFILE_DIR)/$(TEST_DIR)/coverage_raw
MERGED_COVER_DIR:=$(MAKEFILE_DIR)/$(TEST_DIR)/coverage_merged
CODE_COVERAGE:=$(TEST_DIR)/coverage

NAME:=ocne

GIT_COMMIT:=$(shell git rev-parse HEAD)
BUILD_DATE:=$(shell date +"%Y-%m-%dT%H:%M:%SZ")

ifdef RELEASE_VERSION
	CLI_VERSION=${RELEASE_VERSION}
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DIST_DIR:=dist
ENV_NAME=ocne
GO=GO111MODULE=on GOPRIVATE=github.com/oracle-cne/ocne go

export GOCOVERDIR
export BATS_RESULT_DIR

#
# CLI
#

.DEFAULT_GOAL := help
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: run
run:
	$(GO) run ${GOPATH}/src/${OCNE_DIR}/main.go
#
# Go build related tasks
#
$(BUILD_DIR)  \
$(PLATFORM_OUT_DIR) \
$(PLATFORM_INSTRUMENTED_OUT_DIR) :
	mkdir -p $@

$(CHART_EMBED): $(CHART_BUILD_OUT_DIR)
	mkdir -p $@
	cp $(CHART_BUILD_OUT_DIR)/* $@

$(CHART_BUILD_DIR): $(BUILD_DIR)
	git clone -b release/2.0  $(CATALOG_REPO) $@

$(CHART_BUILD_OUT_DIR): $(CHART_BUILD_DIR)
	cd $< && make

.PHONY: build-cli
build-cli: $(CHART_EMBED) $(PLATFORM_OUT_DIR) ## Build CLI for the current system and architecture
	$(GO) build -trimpath -o $(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH) ./...

# Build an instrumented CLI for the current system and architecture
build-cli-instrumented: $(CHARTS_EMBED) $(PLATFORM_INSTRUMENTED_OUT_DIR)
	$(GO) build -cover -trimpath -o $(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented ./...

.PHONY: cli
cli: build-cli ## Build and install the CLI
	cp out/$(shell go env GOOS)_$(shell go env GOARCH)/ocne $(GOPATH)/bin

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./...

# Integration tests
#
# Example commands
#   Run tests only on the setup named "default"
#     make integration-test TEST_PATTERN=default
#
#  Run application tests on the setup named "default"
#    make integration-test TEST_PATTERN=default TEST_FILTERS="--filter-tags APPLICATION"
#
#  Run the cluster upgrade tests on all setups, start with Kubernetes 1.29 on each cluster
#    make integration-test TEST_FILTERS="--filter-tags CLUSTER_UPGRADE" TEST_K8S_VERSION="1.29"
#
$(GOCOVERDIR) $(MERGED_COVER_DIR) $(BATS_RESULT_DIR):
	mkdir -p $@

.PHONY: integration-test
integration-test: $(GOCOVERDIR) $(MERGED_COVER_DIR) $(BATS_RESULT_DIR) build-cli-instrumented
	cd test && PATH="$(MAKEFILE_DIR)/$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented:$$PATH" ./run-tests.sh '$(TEST_PATTERN)'
	$(GO) tool covdata merge -i=$(GOCOVERDIR) -o=$(MERGED_COVER_DIR)
	$(GO) tool covdata textfmt -i=$(MERGED_COVER_DIR) -o=$(CODE_COVERAGE)
	echo To view coverage data, execute \"go tool cover -html=$(CODE_COVERAGE)\"

.PHONY: capi-test
capi-test: $(GOCOVERDIR) $(MERGED_COVER_DIR) $(BATS_RESULT_DIR) build-cli-instrumented
	cd test && PATH="$(MAKEFILE_DIR)/$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented:$$PATH" ./run-tests.sh '$(TEST_PATTERN)' 1
	$(GO) tool covdata merge -i=$(GOCOVERDIR) -o=$(MERGED_COVER_DIR)
	$(GO) tool covdata textfmt -i=$(MERGED_COVER_DIR) -o=$(CODE_COVERAGE)
	echo To view coverage data, execute \"go tool cover -html=$(CODE_COVERAGE)\"

.PHONY: release-test
release-test: $(GOCOVERDIR) $(MERGED_COVER_DIR) $(BATS_RESULT_DIR) build-cli-instrumented
	cd test && PATH="$(MAKEFILE_DIR)/$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented:$$PATH" ./run-tests.sh '$(TEST_PATTERN)' 1 1
	$(GO) tool covdata merge -i=$(GOCOVERDIR) -o=$(MERGED_COVER_DIR)
	$(GO) tool covdata textfmt -i=$(MERGED_COVER_DIR) -o=$(CODE_COVERAGE)
	echo To view coverage data, execute \"go tool cover -html=$(CODE_COVERAGE)\"

clean-charts:
	rm -rf $(CHART_EMBED)
	rm -rf $(CHART_GIT_DIR)

clean: clean-charts ## Delete output from prior builds
	rm -rf $(BUILD_DIR)
	rm -rf $(OUT_DIR)
