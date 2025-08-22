#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

PROJECT_ROOT := $(shell git rev-parse --show-toplevel)
MAIN_PATH    := ./cmd/pixiu
TARGET_NAME  := dubbo-go-pixiu
ifeq ($(shell go env GOOS), windows)
    TARGET_NAME := $(TARGET_NAME).exe
endif

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")

API_CONFIG_PATH ?= configs/api_config.yaml
CONFIG_PATH     ?= configs/conf.yaml

IMAGE_NAME ?= dubbogopixiu/dubbo-go-pixiu

LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)'"

.PHONY: all build run image test integrate-test clean license-check license-check-util import-format check-import-format help

all: build

build: ## build binary
	@echo "==> Building $(TARGET_NAME) version $(VERSION)..."
	@go build $(LDFLAGS) -o $(TARGET_NAME) $(MAIN_PATH)
	@echo "==> Build complete: $(PROJECT_ROOT)/$(TARGET_NAME)"

run: build ## start Pixiu Gateway
	@echo "==> Starting gateway with config: $(CONFIG_PATH)..."
	@./$(TARGET_NAME) gateway start -c $(CONFIG_PATH)

image: ## build Docker image
	@echo "==> Building Docker image $(IMAGE_NAME) with tags: latest, $(VERSION)..."
	@docker build \
		-t $(IMAGE_NAME):latest \
		-t $(IMAGE_NAME):$(VERSION) \
		--build-arg version=$(VERSION) \
		-f Dockerfile --platform linux/amd64 .

test: ## run unit tests
	@echo "==> Running unit tests..."
	@sh before_ut.sh
	@go test ./pkg/... -coverprofile=coverage.txt -covermode=atomic

integrate-test: ## run integration tests
	@echo "==> Running integration tests..."
	@sh start_integrate_test.sh

clean: ## clean up build artifacts
	@echo "==> Cleaning up..."
	@rm -f $(TARGET_NAME) coverage.txt

license-check-util: ## install license header checker utility
	@go install github.com/lsm-dev/license-header-checker/cmd/license-header-checker@latest

license-check: ## check license headers
	@echo "==> Checking license headers..."
	@license-header-checker -v -a -r -i vendor -i .github/actions /tmp/tools/license/license.txt . go

import-format: check-import-format ## format go import blocks
	@echo "==> Formatting Go imports..."
	@imports-formatter -bl=false -module=github.com/apache/dubbo-go-pixiu

check-import-format: ## check installation of imports-formatter
	@command -v imports-formatter >/dev/null 2>&1 || \
		(echo "Error: imports-formatter is not installed. Please run 'go install github.com/dubbogo/tools/cmd/imports-formatter@latest'" && exit 1)


help: ## display this help information
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT:
	@echo "==> Forwarding target '$@' to Makefile.core.mk..."
	@./common/scripts/run.sh make --no-print-directory -e -f Makefile.core.mk $@