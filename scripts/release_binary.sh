#!/usr/bin/env bash

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${PROJECT_PATH}/dist
# shellcheck disable=SC2046
APP_VERSION=$(if [ `git rev-parse --abbrev-ref HEAD` == 'HEAD' ];then git describe --tags HEAD ; else echo "dev" ; fi)

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
cp ./bin/bitxhub "${BUILD_PATH}"/bitxhub
cp ./internal/plugins/build/*.so "${BUILD_PATH}"
if [ "$(uname)" == "Darwin" ]; then
  cd "${BUILD_PATH}" && cp "${PROJECT_PATH}"/build/wasm/lib/darwin-amd64/libwasmer.dylib .
  tar -zcvf bitxhub_darwin_x86_64_"${APP_VERSION}".tar.gz ./bitxhub ./raft.so ./solo.so ./libwasmer.dylib
else
  cd "${BUILD_PATH}" && cp "${PROJECT_PATH}"/build/wasm/lib/linux-amd64/libwasmer.so .
  tar zcvf bitxhub_linux-amd64_"${APP_VERSION}".tar.gz ./bitxhub ./*.so
fi
