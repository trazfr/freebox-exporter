# Copyright (C) 2021 Nicolas Lamirault <nicolas.lamirault@gmail.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

APP = freebox-exporter
BANNER = B B O X  E X P O R T E R

# Needs to be defined before including Makefile.common to auto-generate targets
DOCKER_ARCHS ?= amd64 armv7 arm64

include Makefile.common

DOCKER_IMAGE_NAME ?= freebox-exporter

# VERSION=$(shell \
# 	grep "const Version" version/version.go \
# 	|awk -F'=' '{print $$2}' \
# 	|sed -e "s/[^0-9.]//g" \
# 	|sed -e "s/ //g")

DEBUG ?=
SHELL = /bin/bash

DIR = $(shell pwd)

DOCKER = docker

GO = go

# GOX = gox -os="linux darwin windows freebsd openbsd netbsd"
GOX = gox -osarch="linux/amd64" -osarch="linux/arm64" -osarch="linux/arm" -osarch="darwin/amd64" -osarch="windows/amd64"
GOX_ARGS = "-output={{.Dir}}-$(VERSION)_{{.OS}}_{{.Arch}}"

MAIN = github.com/nlamirault/freebox-exporter

PACKAGE=$(APP)-$(VERSION)
ARCHIVE=$(PACKAGE).tar

EXE = $(shell ls freebox-exporter-*)

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
INFO_COLOR=\033[36m
WHITE_COLOR=\033[1m

MAKE_COLOR=\033[33;01m%-20s\033[0m

.DEFAULT_GOAL := help

OK=[✅]
KO=[❌]
WARN=[⚠️]

.PHONY: help
help:
	@echo -e "$(OK_COLOR)                  $(BANNER)$(NO_COLOR)"
	@echo "------------------------------------------------------------------"
	@echo ""
	@echo -e "${ERROR_COLOR}Usage${NO_COLOR}: make ${INFO_COLOR}<target>${NO_COLOR}"
	@awk 'BEGIN {FS = ":.*##"; } /^[a-zA-Z0-9_-]+:.*?##/ { printf "  ${INFO_COLOR}%-30s${NO_COLOR} %s\n", $$1, $$2 } /^##@/ { printf "\n${WHITE_COLOR}%s${NO_COLOR}\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

guard-%:
	@if [ "${${*}}" = "" ]; then \
		echo -e "$(ERROR_COLOR)Environment variable $* not set$(NO_COLOR)"; \
		exit 1; \
	fi

check-%:
	@if $$(hash $* 2> /dev/null); then \
		echo -e "$(OK_COLOR)$(OK)$(NO_COLOR) $*"; \
	else \
		echo -e "$(ERROR_COLOR)$(KO)$(NO_COLOR) $*"; \
	fi

print-%:
	@if [ "${$*}" == "" ]; then \
		echo -e "$(ERROR_COLOR)[KO]$(NO_COLOR) $* = ${$*}"; \
	else \
		echo -e "$(OK_COLOR)[OK]$(NO_COLOR) $* = ${$*}"; \
	fi

# ====================================
# D E V E L O P M E N T
# ====================================

##@ Development

.PHONY: clean
clean: ## Cleanup
	@echo -e "$(OK_COLOR)[$(APP)] Cleanup$(NO_COLOR)"
	@rm -fr $(EXE) $(APP) $(APP)-*.tar.gz

.PHONY: validate
validate: ## Execute git-hooks
	@pre-commit run -a

# .PHONY: build
# build: ## Make binary
# 	@echo -e "$(OK_COLOR)[$(APP)] Build $(NO_COLOR)"
# 	@$(GO) build .

# .PHONY: test
# test: ## Launch unit tests
# 	@echo -e "$(OK_COLOR)[$(APP)] Launch unit tests $(NO_COLOR)"
# 	@$(GO) test

# .PHONY: run
# run: ## Start exporter
# 	@echo -e "$(OK_COLOR)[$(APP)] Start the exporter $(NO_COLOR)"
# 	@./$(APP) -log.level DEBUG

# ====================================
# D O C K E R
# ====================================

##@ Docker

docker-build: guard-VERSION ## Build Docker image
	@echo -e "$(OK_COLOR)Docker build $(APP):$(VERSION)$(NO_COLOR)"
	@DOCKER_BUILDKIT=1 docker build -t $(APP):$(VERSION) .

docker-run: guard-VERSION ## Run the Docker image
	@echo -e "$(OK_COLOR)Docker run $(APP):$(VERSION)$(NO_COLOR)"
	@docker run --rm=true $(APP):$(VERSION) /freebox-exporter --help
