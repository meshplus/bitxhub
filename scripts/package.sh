#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
CONFIG_PATH=${PROJECT_PATH}/config
PACKAGE_PATH=${CURRENT_PATH}/package
APP_NAME=axiom

BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_red() {
    printf "${RED}%s${NC}\n" "$1"
}

function prepare() {
  print_blue "===> Generating ${APP_NAME} package"
  root=${PACKAGE_PATH}

  cp ${PROJECT_PATH}/bin/${APP_NAME} ${root}
}

function package() {
    print_blue "=== Package node and configuration"
    root=${PACKAGE_PATH}

    cd ${root}
    if [ -n "$VERSION" ]; then
        echo "=== version is ${VERSION}"
        tar -zcvf axiom-${VERSION}.tar.gz ${APP_NAME} *.sh
    else
        tar -zcvf axiom-dev.tar.gz ${APP_NAME} *.sh
    fi
    print_blue "=== Package is under path: ${root}"
}

prepare
package
