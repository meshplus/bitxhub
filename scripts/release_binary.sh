#!/usr/bin/env bash

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${PROJECT_PATH}/build
APP_VERSION=${1:-'v1.6.0'}

print_blue "===> 1. Install packr"
if ! type packr >/dev/null 2>&1; then
  go get -u github.com/gobuffalo/packr/packr
fi

print_blue "===> 2. build bitxhub"
cd "${PROJECT_PATH}" && make build

print_blue "===> 3. build plugins: raft, solo"
cd "${PROJECT_PATH}"/internal/plugins && make plugins

print_blue "===> 4. pack binarys"
cd "${PROJECT_PATH}"
cp ./bin/bitxhub ./build/bitxhub
cp ./internal/plugins/build/*.so ./build/
if [ "$(uname)" == "Darwin" ]; then
    cd "${BUILD_PATH}" && tar zcvf bitxhub_macos_x86_64_v"${APP_VERSION}".tar.gz ./bitxhub ./raft.so ./solo.so ./libwasmer.dylib
    mv ./*.tar.gz ../dist/
else
    cd "${BUILD_PATH}" && tar zcvf bitxhub_linux-amd64_v"${APP_VERSION}".tar.gz ./bitxhub ./*.so
    mv ./*.tar.gz ../dist/
fi
