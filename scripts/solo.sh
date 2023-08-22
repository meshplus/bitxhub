#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
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
  print_blue "===> Start solo axiom"
  rm -rf "${BUILD_PATH}"
  axiom --repo="${BUILD_PATH}" config generate --solo
  if [ -n "$TAGS" ]; then
    if [ "$TAGS" = "mockConsensus" ] || [ "$TAGS" = "mockExecutor" ]; then
      axiom --repo="${BUILD_PATH}" start --mode="${TAGS}"
    else
      print_red "TAGS should be mockConsensus or mockExecutor"
      exit 1
    fi
  else
    axiom --repo="${BUILD_PATH}" start
  fi
}

start
