#!/usr/bin/env bash

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${PROJECT_PATH}/build
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
cp ./bin/bitxhub ./build/bitxhub
cp ./internal/plugins/build/*.so ./build/
if [ "$(uname)" == "Darwin" ]; then
    cd "${BUILD_PATH}" && tar zcvf bitxhub_"${APP_VERSION}"_Darwin_x86_64.tar.gz ./bitxhub ./raft.so ./solo.so ./libwasmer.dylib
    mv ./*.tar.gz ../dist/
else
    cd "${BUILD_PATH}" && tar zcvf bitxhub_"${APP_VERSION}"_Linux-amd64.tar.gz ./bitxhub ./*.so
    mv ./*.tar.gz ../dist/
fi
