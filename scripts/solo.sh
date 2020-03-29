#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build_solo

function prepare() {
  rm -rf "${BUILD_PATH}"
  mkdir -p "${BUILD_PATH}"
}

function config() {
  mkdir -p ${BUILD_PATH}/certs
  cp -r "${CURRENT_PATH}"/certs/node1/certs/* "${BUILD_PATH}"/certs
}

function compile() {
  cd "${PROJECT_PATH}"
  make install

  ## build plugin
  cd "${PROJECT_PATH}"/internal/plugins
  make solo
}

function start() {
  bitxhub --repo="${BUILD_PATH}" init
  mkdir -p "${BUILD_PATH}"/plugins
  cp "${PROJECT_PATH}"/internal/plugins/build/solo.so "${BUILD_PATH}"/plugins/
  bitxhub --repo="${BUILD_PATH}" start
}

prepare
config
compile
start
