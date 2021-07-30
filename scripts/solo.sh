#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build_solo

function prepare() {
  rm -rf "${BUILD_PATH}"
  mkdir -p "${BUILD_PATH}"
}

function config() {
  mkdir -p "${BUILD_PATH}"/certs
  cp -r "${CURRENT_PATH}"/certs/node1/* "${BUILD_PATH}"/
}

function compile() {
  print_blue "===> Compileing bitxhub"
  cd "${PROJECT_PATH}"
  make install${TAGS}

  ## build plugin
  cd "${PROJECT_PATH}"/internal/plugins
  make solo${TAGS}
}

function start() {
  print_blue "===> Start solo bitxhub"
  bitxhub --repo="${BUILD_PATH}" init
  bitxhubConfig=${BUILD_PATH}/bitxhub.toml
  x_replace "s/solo = false/solo = true/g" "${bitxhubConfig}"
  x_replace "s/raft.so/solo.so/g" "${bitxhubConfig}"
  mkdir -p "${BUILD_PATH}"/plugins
  cp "${PROJECT_PATH}"/internal/plugins/build/solo.so "${BUILD_PATH}"/plugins/
  bitxhub --repo="${BUILD_PATH}" start
}

prepare
config
compile
start
