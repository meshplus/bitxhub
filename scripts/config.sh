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
N=$1

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
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
  cd "${PROJECT_PATH}"
  make build

  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"

  cd "${PROJECT_PATH}"/internal/plugins
  make raft
}

function generate() {
  cd "${BUILD_PATH}"
  cp "${PROJECT_PATH}"/bin/bitxhub "${BUILD_PATH}"
  cp -rf "${PROJECT_PATH}"/internal/plugins/build/raft.so "${BUILD_PATH}"

  "${BUILD_PATH}"/bitxhub cert ca
  "${BUILD_PATH}"/bitxhub cert priv gen --name agency
  "${BUILD_PATH}"/bitxhub cert csr --key ./agency.priv --org Agency
  "${BUILD_PATH}"/bitxhub cert issue --key ./ca.priv --cert ./ca.cert --csr ./agency.csr --is_ca true
  rm agency.csr

  for ((i = 1; i < N + 1; i = i + 1)); do
    repo=${BUILD_PATH}/node${i}
    mkdir -p "${repo}"
   "${BUILD_PATH}"/bitxhub --repo="${repo}" init

    mkdir -p "${repo}"/plugins
    mkdir -p "${repo}"/certs

    cd "${repo}"/certs
    "${BUILD_PATH}"/bitxhub cert priv gen --name node
    "${BUILD_PATH}"/bitxhub cert csr --key ./node.priv --org Node${i}
    "${BUILD_PATH}"/bitxhub cert issue --key "${BUILD_PATH}"/agency.priv --cert "${BUILD_PATH}"/agency.cert --csr ./node.csr
    "${BUILD_PATH}"/bitxhub key gen --name key
    cp "${BUILD_PATH}"/ca.cert "${repo}"/certs
    cp "${BUILD_PATH}"/agency.cert "${repo}"/certs
    rm "${repo}"/certs/node.csr

    id=$("${BUILD_PATH}"/bitxhub --repo="${repo}" cert priv pid --path "${repo}"/certs/node.priv)
    addr=$("${BUILD_PATH}"/bitxhub --repo="${repo}" key address --path "${repo}"/certs/key.priv)

    echo "${id}" >>"${BUILD_PATH}"/pids
    echo "${addr}" >>"${BUILD_PATH}"/addresses

    echo "#!/usr/bin/env bash" >"${repo}"/start.sh
    echo "./bitxhub --repo \$(pwd)" start >>"${repo}"/start.sh
  done
}

function printHelp() {
  print_blue "Usage:  "
  echo "  config.sh <number>"
  echo "    <number> - node number"
  echo "  config.sh -h (print this message)"
}

if [ ! $1 ]; then
  printHelp
  exit 1
fi

prepare
generate
