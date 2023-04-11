#!/usr/bin/env bash

set -x

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
CONFIG_PATH=${PROJECT_PATH}/config
BUILD_PATH=${CURRENT_PATH}/build
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'
N=4

print_blue "===> Clean $N nodes storage"
  for ((i = 1; i < N + 1; i = i + 1)); do
    if [ -d "${BUILD_PATH}/node$i/storage" ]; then
      rm -rf "${BUILD_PATH}/node$i/storage"
    fi
  done