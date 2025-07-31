# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
include make/quality.mk

GOPATH ?= $(shell go env GOPATH)

CATALOG_REPO=https://github.com/oracle-cne/catalog.git
MAKEFILE_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
INFO_DIR:=github.com/oracle-cne/ocne/cmd/info
CLONE_DIR:=${MAKEFILE_DIR}/temp-clone-dir
BUILD_DIR:=build
OUT_DIR:=out
PLATFORM_OUT_DIR:=$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)
PLATFORM_INSTRUMENTED_OUT_DIR:=$(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented 
CHART_BUILD_DIR:=$(BUILD_DIR)/catalog
CHART_BUILD_OUT_DIR:=$(CHART_BUILD_DIR)/repo
CHART_GIT_DIR:=build/charts

CHART_EMBED:=pkg/catalog/embedded/charts

CATALOG_BRANCH?=release/2.2

DEVELOPER_CHART_BUILD_DIR:=${BUILD_DIR}/developer-catalog
DEVELOPER_CATALOG_BRANCH?=developer

TEST_DIR:=$(OUT_DIR)/tests
GOCOVERDIR:=$(MAKEFILE_DIR)/$(TEST_DIR)/coverage_raw
MERGED_COVER_DIR:=$(MAKEFILE_DIR)/$(TEST_DIR)/coverage_merged
CODE_COVERAGE:=$(TEST_DIR)/coverage

NAME:=ocne

GIT_COMMIT:=$(shell git rev-parse HEAD)
BUILD_DATE:=$(shell date +"%Y-%m-%dT%H:%M:%SZ")
CLI_VERSION:=$(shell grep Version: ${MAKEFILE_DIR}/buildrpm/ocne.spec | cut -d ' ' -f 2)-$(shell grep Release: ${MAKEFILE_DIR}/buildrpm/ocne.spec | cut -d ' ' -f 2 | cut -d '%' -f 1)
OS:=$(shell uname)
ifeq ($(OS), Linux)
	CLI_VERSION=$(shell rpmspec -q --queryformat='%{VERSION}-%{RELEASE}' ${MAKEFILE_DIR}/buildrpm/ocne.spec)
endif
ifndef RELEASE_BRANCH
	RELEASE_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
endif

DEVELOPER_BUILD?=
ifeq ($(DEVELOPER_BUILD),)
	# Default the value based on the branch name.  If the branch name is prefixed
	# with "release/" then set to false, otherwise true.
	ifeq ($(findstring release/,$(RELEASE_BRANCH)), release/)
		DEVELOPER_BUILD=false
	else
		# Disable all developer builds by default, there is currently no
		# content on the developer branch of the ocne-catalog repo.
		DEVELOPER_BUILD=false
	endif
endif

DIST_DIR:=dist
ENV_NAME=ocne
GO=GO111MODULE=on GOPRIVATE=github.com/oracle-cne/ocne go

CLI_GO_LDFLAGS=-X '${INFO_DIR}.gitCommit=${GIT_COMMIT}' -X '${INFO_DIR}.buildDate=${BUILD_DATE}' -X '${INFO_DIR}.cliVersion=${CLI_VERSION}'
CLI_BUILD_TAGS=
ifeq (${DEVELOPER_BUILD}, true)
	CLI_BUILD_TAGS=-tags developer
endif

export GOCOVERDIR

#
# CLI
#

.DEFAULT_GOAL := help
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: run
run:
	$(GO) run -trimpath -ldflags "${CLI_GO_LDFLAGS}" ${CLI_BUILD_TAGS} ./...
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

$(CHART_BUILD_DIR):
	mkdir -p $(BUILD_DIR)
	git clone -b ${CATALOG_BRANCH}  $(CATALOG_REPO) $@

$(CHART_BUILD_OUT_DIR): $(CHART_BUILD_DIR) ${DEVELOPER_CHART_BUILD_DIR}
ifeq (${DEVELOPER_BUILD},true)
	cp -r ${DEVELOPER_CHART_BUILD_DIR}/charts/* ${CHART_BUILD_DIR}/charts
	cp ${DEVELOPER_CHART_BUILD_DIR}/olm/icons/* ${CHART_BUILD_DIR}/olm/icons
	cd $< && SUPPORT_MATRIX_CHECKS=false make
else
	cd $< && make
endif

$(DEVELOPER_CHART_BUILD_DIR):
ifeq (${DEVELOPER_BUILD},true)
	mkdir -p $(BUILD_DIR)
	git clone -b ${DEVELOPER_CATALOG_BRANCH}  $(CATALOG_REPO) $@
endif

.PHONY: build-cli
build-cli: $(CHART_EMBED) $(PLATFORM_OUT_DIR) ## Build CLI for the current system and architecture
	$(GO) build -trimpath -ldflags "${CLI_GO_LDFLAGS}" ${CLI_BUILD_TAGS} -o $(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH) ./...

# Build an instrumented CLI for the current system and architecture
build-cli-instrumented: $(CHART_EMBED) $(PLATFORM_INSTRUMENTED_OUT_DIR)
	$(GO) build -cover -trimpath -ldflags "${CLI_GO_LDFLAGS}" ${CLI_BUILD_TAGS} -o $(OUT_DIR)/$(shell go env GOOS)_$(shell go env GOARCH)_instrumented ./...

.PHONY: cli
cli: build-cli ## Build and install the CLI
	cp out/$(shell go env GOOS)_$(shell go env GOARCH)/ocne $(GOPATH)/bin

.PHONY: unit-test
unit-test: cli
	$(GO) test -v  ./...

clean-charts:
	rm -rf $(CHART_EMBED)
	rm -rf $(CHART_GIT_DIR)

clean: clean-charts ## Delete output from prior builds
	rm -rf $(BUILD_DIR)
	rm -rf $(OUT_DIR)
