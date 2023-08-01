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
  print_blue "===> Compileing axiom"
  cd "${PROJECT_PATH}"
  make install${TAGS}

}

function start() {
  print_blue "===> Start solo axiom"
  axiom --repo="${BUILD_PATH}" init
  cd ${BUILD_PATH} && axiom key gen
  axiomConfig=${BUILD_PATH}/axiom.toml
  x_replace "s/solo = false/solo = true/g" "${axiomConfig}"
  x_replace "s/rbft/solo/g" "${axiomConfig}"
  axiom --repo="${BUILD_PATH}" start
}

prepare
config
start
