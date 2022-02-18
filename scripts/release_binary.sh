#!/usr/bin/env bash

source x.sh
N=4
CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${PROJECT_PATH}/dist
CONFIG_PATH=${PROJECT_PATH}/config
APP_VERSION=$(if [ "$(git rev-parse --abbrev-ref HEAD)" == "HEAD" ]; then git describe --tags HEAD; else echo "dev"; fi)

rm -rf "$BUILD_PATH"
mkdir -p "$BUILD_PATH"
print_blue "===> 1. Install packr"
if ! type packr >/dev/null 2>&1; then
  go get -u github.com/gobuffalo/packr/packr
fi

print_blue "===> 2. Build bitxhub"
cd "${PROJECT_PATH}" && make build

print_blue "===> 3. Pack binary"
cd "${PROJECT_PATH}" || (echo "project path is not exist" && return)
cp ./bin/bitxhub "${BUILD_PATH}"/bitxhub
if [ "$(uname)" == "Darwin" ]; then
  cd "${BUILD_PATH}" && cp "${PROJECT_PATH}"/build/wasm/lib/darwin-amd64/libwasmer.dylib .
  tar -zcvf bitxhub_darwin_x86_64_"${APP_VERSION}".tar.gz ./bitxhub ./libwasmer.dylib
else
  cd "${BUILD_PATH}" && cp "${PROJECT_PATH}"/build/wasm/lib/linux-amd64/libwasmer.so .
  tar -zcvf bitxhub_linux-amd64_"${APP_VERSION}".tar.gz ./bitxhub ./libwasmer.so
fi

if [ "$(uname)" = "Linux" ]; then
  print_blue "===> 4. Generating $N nodes configuration"
  for ((i = 1; i < N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    mkdir -p "${root}"

    cp -rf "${CURRENT_PATH}"/certs/node"${i}"/* "${root}"
    cp -rf "${CONFIG_PATH}"/* "${root}"

    echo " #!/usr/bin/env bash" >"${root}"/start.sh
    echo "./bitxhub --repo \$(pwd)" start >>"${root}"/start.sh

    bitxhubConfig=${root}/bitxhub.toml
    networkConfig=${root}/network.toml

    x_replace "s/60011/6001${i}/g" "${bitxhubConfig}"
    x_replace "s/9091/909${i}/g" "${bitxhubConfig}"
    x_replace "s/53121/5312${i}/g" "${bitxhubConfig}"
    x_replace "s/40011/4001${i}/g" "${bitxhubConfig}"
    x_replace "s/8881/888${i}/g" "${bitxhubConfig}"
    x_replace "1s/1/${i}/" "${networkConfig}"
    tar -zcvf example_bitxhub_"${APP_VERSION}".tar.gz node*
  done
fi
