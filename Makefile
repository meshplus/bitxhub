
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
APP_NAME = bitxhub
APP_VERSION = 1.0.0-rc1

# build with verison infos
VERSION_DIR = github.com/meshplus/${APP_NAME}
BUILD_DATE = $(shell date +%FT%T)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS += -X "${VERSION_DIR}.BuildDate=${BUILD_DATE}"
LDFLAGS += -X "${VERSION_DIR}.CurrentCommit=${GIT_COMMIT}"
LDFLAGS += -X "${VERSION_DIR}.CurrentBranch=${GIT_BRANCH}"
LDFLAGS += -X "${VERSION_DIR}.CurrentVersion=${APP_VERSION}"

GO  = GO111MODULE=on go
TEST_PKGS := $(shell $(GO) list ./... | grep -v 'mock_*' | grep -v 'tester')

RED=\033[0;31m
GREEN=\033[0;32m
BLUE=\033[0;34m
NC=\033[0m

help: Makefile
	@printf "${BLUE}Choose a command run:${NC}\n"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/    /'

## make prepare: Preparation before development
prepare:
	@cd scripts && sh prepare.sh

## make test: Run go unittest
test:
	go generate ./...
	@$(GO) test ${TEST_PKGS} -count=1

## make test-coverage: Test project with cover
test-coverage:
	@$(GO) test -coverprofile cover.out ${TEST_PKGS}
	$(GO) tool cover -html=cover.out -o cover.html

## make tester: Run integration test
tester:
	cd tester && $(GO) test -v -run TestTester

## make install: Go install the project
install:
	cd internal/repo && packr
	$(GO) install -ldflags '${LDFLAGS}' ./cmd/${APP_NAME}
	@printf "${GREEN}Build bitxhub successfully!${NC}\n"

build:
	cd internal/repo && packr
	@mkdir -p bin
	$(GO) build -ldflags '${LDFLAGS}' ./cmd/${APP_NAME}
	@mv ./bitxhub bin
	@printf "${GREEN}Build bitxhub successfully!${NC}\n"

## make linter: Run golanci-lint
linter:
	golangci-lint run

## make cluster: Run cluster including 4 nodes
cluster:
	@cd scripts && bash cluster.sh

.PHONY: tester
