#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
N=$1

function build_config() {
  bash config.sh "${N}"
}

function build_plugins() {
  ## build plugin(make plugin type=<rbft> or <raft>)
  cd "${PROJECT_PATH}"/internal/plugins
  make raft

  for ((i = 1; i < N + 1; i = i + 1)); do
    mkdir -p "${BUILD_PATH}"/node${i}
    cp -rf "${PROJECT_PATH}"/internal/plugins/build "${BUILD_PATH}"/node${i}/plugins
  done
}

build_config
build_plugins
