#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build_solo

function start() {
  print_blue "===> Start solo axiom"
  rm -rf "${BUILD_PATH}"
  axiom --repo="${BUILD_PATH}" config generate --solo
  axiom --repo="${BUILD_PATH}" start
}

start
