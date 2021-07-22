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
  cp -r "${CURRENT_PATH}"/certs/node1/certs/* "${BUILD_PATH}"/certs
}

function compile() {
  print_blue "===> Compileing bitxhub"
  cd "${PROJECT_PATH}"
  make install${TAGS}

}

function start() {
  print_blue "===> Start solo bitxhub"
  bitxhub --repo="${BUILD_PATH}" init
  cd ${BUILD_PATH} && bitxhub key gen
  bitxhubConfig=${BUILD_PATH}/bitxhub.toml
  x_replace "s/solo = false/solo = true/g" "${bitxhubConfig}"
  x_replace "s/raft/solo/g" "${bitxhubConfig}"
  bitxhub --repo="${BUILD_PATH}" start
}

prepare
config
compile
start
