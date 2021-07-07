set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build


# help prompt message
function printHelp() {
  print_blue "Usage:  "
  echo "  build_release.sh [-n <node_num>] [-v <bitxhub_version>]"
  echo "  -'n' - node number to be deployed in one server"
  echo "  -'v' - the version of bitxhub checkout"
  echo "  build_release.sh -h (print this message)"
}

function build_release() {
  git checkout "$APP_VERSION"
  print_blue "Generate config"
  bash config.sh "$NODE_NUM"
  build_linux

  bash config.sh "$NODE_NUM"
  build_darwin
}

function build_linux() {
  print_blue "Compile bitxhub_linux-amd64_${APP_VERSION}"
  bash cross_compile.sh linux-amd64 "${PROJECT_PATH}"

  ## prepare deploy package
  cd "${CURRENT_PATH}"
  cp ../bin/bitxhub_linux-amd64 "${BUILD_PATH}"/bitxhub
  cp ../build/libwasmer.so "${BUILD_PATH}"/
  tar zcf build_linux-amd64_"${APP_VERSION}".tar.gz build
}

function build_darwin() {
  print_blue "Compile bitxhub_macos_x86_64_${APP_VERSION}"
  cd "${PROJECT_PATH}"
  make build

  ## prepare deploy package
  cd "${CURRENT_PATH}"
  cp ../bin/bitxhub "${BUILD_PATH}"/bitxhub
  cp ../build/libwasmer.dylib "${BUILD_PATH}"/
  tar zcf build_macos_x86_64_"${APP_VERSION}".tar.gz build
  tar zcf bitxhub_macos_x86_64_"${APP_VERSION}".tar.gz build/bitxhub build/*.so build/libwasmer.dylib
}

NODE_NUM=4
APP_VERSION=v1.0.0-rc1

while getopts "h?n:v:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  n)
    NODE_NUM=$OPTARG
    ;;
  v)
    APP_VERSION=$OPTARG
    ;;
  esac
done

build_release
