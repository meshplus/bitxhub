
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
APP_NAME = axiom
export GODEBUG=x509ignoreCN=0

# build with verison infos
VERSION_DIR = github.com/axiomesh/${APP_NAME}
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
TEST_PKGS2 := $(shell $(GO) list ./... | grep -v 'syncer' | grep -v 'peermgr'| grep -v 'vm' | grep -v 'proof'  | grep -v 'appchain' | grep -v 'repo' | grep -v 'mock_*' | grep -v 'tester' | grep -v 'proto' | grep -v 'cmd'| grep -v 'api')

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
test: prepare
	go generate ./...
	@$(GO) test -timeout 300s ${TEST_PKGS} -count=1

## make test-coverage: Test project with cover
test-coverage: prepare
	go generate ./...
	@go test -timeout 300s -short -coverprofile cover.out -covermode=atomic ${TEST_PKGS2}
	@cat cover.out | grep -v "pb.go" >> coverage.txt

## make smoke-test: Run smoke test
smoke-test: prepare
	cd scripts && bash smoke_test.sh -b ${BRANCH}

## make install: Go install the project
install: prepare
	cd internal/repo && packr2
	$(GO) install -ldflags '${GOLDFLAGS}' ./cmd/${APP_NAME}
	@printf "${GREEN}Install ${APP_NAME} successfully!${NC}\n"

## make build: Go build the project
build: prepare
	cd internal/repo && packr2
	@mkdir -p bin
	$(GO) build -ldflags '${GOLDFLAGS}' ./cmd/${APP_NAME}
	@mv ./${APP_NAME} bin
	@printf "${GREEN}Build ${APP_NAME} successfully!${NC}\n"

## make release-binary: Build release before push
release-binary:
	@cd scripts && bash release_binary.sh

## make linter: Run golanci-lint
linter:
	golangci-lint run --timeout=5m --new-from-rev=HEAD~1 -v

## make cluster: Run cluster including 4 nodes
cluster:install${TAGS}
	@cd scripts && bash cluster.sh TAGS=${TAGS}

## make solo: Run one node in solo mode
solo:install${TAGS}
	@cd scripts && bash solo.sh TAGS=${TAGS}

precommit: test-coverage linter

.PHONY: build