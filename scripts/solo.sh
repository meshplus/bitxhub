#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build_solo

BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_red() {
    printf "${RED}%s${NC}\n" "$1"
}

function start() {
  print_blue "===> Start solo axiom-ledger"
  rm -rf "${BUILD_PATH}" && mkdir ${BUILD_PATH}
  cp -rf ${CURRENT_PATH}/package/* ${BUILD_PATH}/
  cp -f ${PROJECT_PATH}/bin/axiom-ledger ${BUILD_PATH}/tools/bin/
  ${BUILD_PATH}/axiom-ledger config generate --solo
  ${BUILD_PATH}/axiom-ledger start
}

start
