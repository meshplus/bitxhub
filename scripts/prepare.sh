#!/usr/bin/env bash

BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function go_install() {
  version=$(go env GOVERSION)
  if [[ ! "$version" < "go1.16" ]];then
      go install "$@"
  else
      go get "$@"
  fi
}

if ! type golangci-lint >/dev/null 2>&1; then
  print_blue "===> 2. Install golangci-lint"
  version=$(go env GOVERSION)
  if [[ ! "$version" < "go1.16" ]];then
      go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3
  else
      go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.23.0
  fi
fi

if ! type mockgen >/dev/null 2>&1; then
  print_blue "===> 3. Install mockgen"
  go_install github.com/golang/mock/mockgen@v1.6.0
fi