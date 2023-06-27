
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
APP_NAME = bitxhub
export GODEBUG=x509ignoreCN=0

# build with verison infos
VERSION_DIR = github.com/meshplus/${APP_NAME}
BUILD_DATE = $(shell date +%FT%T)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
ifeq (${GIT_BRANCH},HEAD)
  APP_VERSION = $(shell git describe --tags HEAD)
else
  APP_VERSION = dev
endif

GOLDFLAGS += -X "${VERSION_DIR}.BuildDate=${BUILD_DATE}"
GOLDFLAGS += -X "${VERSION_DIR}.CurrentCommit=${GIT_COMMIT}"
GOLDFLAGS += -X "${VERSION_DIR}.CurrentBranch=${GIT_BRANCH}"
GOLDFLAGS += -X "${VERSION_DIR}.CurrentVersion=${APP_VERSION}"

GO  = GO111MODULE=on go
TEST_PKGS := $(shell $(GO) list ./... | grep -v 'mock_*' | grep -v 'tester' | grep -v 'proto' | grep -v 'cmd'| grep -v 'api')
TEST_PKGS2 := $(shell $(GO) list ./... | grep -v 'etcdraft' | grep -v 'syncer' | grep -v 'peermgr'| grep -v 'vm' | grep -v 'proof'  | grep -v 'appchain' | grep -v 'repo' | grep -v 'mock_*' | grep -v 'tester' | grep -v 'proto' | grep -v 'cmd'| grep -v 'api')

RED=\033[0;31m
GREEN=\033[0;32m
BLUE=\033[0;34m
NC=\033[0m

help: Makefile
	@printf "${BLUE}Choose a command run:${NC}\n"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/    /'

## make prepare: Preparation before development
prepare:
	@cd scripts && bash prepare.sh

## make test: Run go unittest
test:
	go generate ./...
	@$(GO) test -timeout 300s ${TEST_PKGS} -count=1

## make test-coverage: Test project with cover
test-coverage:
	go generate ./...
	@go test -timeout 300s -short -coverprofile cover.out -covermode=atomic ${TEST_PKGS2}
	@cat cover.out | grep -v "pb.go" >> coverage.txt

## make tester: Run integration test
tester:
	go generate ./...
	cd tester && $(GO) test -v -run TestTester

## make install: Go install the project
install:
	cd internal/repo && packr
	$(GO) install -ldflags '${GOLDFLAGS}' ./cmd/${APP_NAME}
	@printf "${GREEN}Install bitxhub successfully!${NC}\n"

## make build: Go build the project
build:
	cd internal/repo && packr
	@mkdir -p bin
	$(GO) build -ldflags '${GOLDFLAGS}' ./cmd/${APP_NAME}
	@mv ./bitxhub bin
	@printf "${GREEN}Build bitxhub successfully!${NC}\n"

## make release: Build release before push
release-binary:
	@cd scripts && bash release_binary.sh

## make linter: Run golanci-lint
linter:
	golangci-lint run

## make cluster: Run cluster including 4 nodes
cluster:install${TAGS}
	@cd scripts && bash cluster.sh TAGS=${TAGS}

## make solo: Run one node in solo mode
solo:install${TAGS}
	@cd scripts && bash solo.sh TAGS=${TAGS}

.PHONY: tester build
