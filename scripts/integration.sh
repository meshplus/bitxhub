#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
CONFIG_PATH=${PROJECT_PATH}/config
BUILD_PATH=${CURRENT_PATH}/build
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'
N=4

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

# The sed commend with system judging
# Examples:
# sed -i 's/a/b/g' bob.txt => x_replace 's/a/b/g' bob.txt
function x_replace() {
  system=$(uname)

  if [ "${system}" = "Linux" ]; then
    sed -i "$@"
  else
    sed -i '' "$@"
  fi
}

function prepare() {
  print_blue "===> Generating $N nodes configuration"
  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"
  for ((i = 1; i < N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    mkdir -p "${root}"

    cp -rf "${CURRENT_PATH}"/certs/node${i}/certs "${root}"
    cp -rf "${CONFIG_PATH}"/* "${root}"

    echo " #!/usr/bin/env bash" >"${root}"/start.sh
    echo "./bitxhub --root \$(pwd)" start >>"${root}"/start.sh

    bitxhubConfig=${root}/bitxhub.toml
    networkConfig=${root}/network.toml
    x_replace "s/60011/6001${i}/g" "${bitxhubConfig}"
    x_replace "s/9091/909${i}/g" "${bitxhubConfig}"
    x_replace "s/53121/5312${i}/g" "${bitxhubConfig}"
    x_replace "s/9091/909${i}/g" "${root}"/api
    x_replace "1s/1/${i}/" "${networkConfig}"
  done
}

function compile() {
  print_blue "===> Compiling bitxhub"
  cd "${PROJECT_PATH}"
  make install
}

prepare
compile

bitxhub version
cd "${CURRENT_PATH}"
for ((i = 1; i < N + 1; i = i + 1)); do
  echo "Start node${i}"
  nohup bitxhub --repo="${BUILD_PATH}"/node${i} start &
done

