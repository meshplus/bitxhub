#!/usr/bin/env bash

set -e

# ./scripts/update_deps.sh tag_name/branch_name

CURRENT_PATH=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
cd ${PROJECT_PATH}

GOPROXY=direct go get github.com/axiomesh/axiom-kit@$1
GOPROXY=direct go get github.com/axiomesh/axiom-bft@$1
GOPROXY=direct go get github.com/axiomesh/axiom-p2p@$1
GOPROXY=direct go get github.com/axiomesh/eth-kit@$1
go mod tidy