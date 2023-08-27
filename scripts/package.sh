#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
CONFIG_PATH=${PROJECT_PATH}/config
PACKAGE_PATH=${CURRENT_PATH}/package

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
  print_blue "===> Generating nodes configuration"
  rm -rf "${PACKAGE_PATH}"
  root=${PACKAGE_PATH}

  axiom --repo="${root}" config generate

  echo " #!/usr/bin/env bash" >"${root}"/start.sh
  echo "./axiom --repo \$(pwd)" start >>"${root}"/start.sh
  cp ${PROJECT_PATH}/bin/axiom ${root}
  cp ${PROJECT_PATH}/scripts/restart.sh ${root}
  cp ${PROJECT_PATH}/scripts/version.sh ${root}
  
  axiomConfig=${root}/axiom.toml
  networkConfig=${root}/network.toml
}

function package() {
    print_blue "=== Package node and configuration"
    root=${PACKAGE_PATH}

    cd ${root}
    if [ -n "$VERSION" ]; then
        echo "=== version is ${VERSION}"
        tar -zcvf axiom-${VERSION}.tar.gz *
    else
        tar -zcvf axiom-dev.tar.gz *
    fi
    print_blue "=== Package is under path: ${root}"
}

prepare
package
