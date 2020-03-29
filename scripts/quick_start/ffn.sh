#!/usr/bin/env bash

set -e

VERSION=1.0
CURRENT_PATH=$(pwd)
FABRIC_SAMPLE_PATH=${CURRENT_PATH}/fabric-samples
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

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

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function printHelp() {
  print_blue "Usage:  "
  echo "  ffn.sh <mode>"
  echo "    <mode> - one of 'up', 'down', 'restart'"
  echo "      - 'up' - bring up the fabric first network"
  echo "      - 'down' - clear the fabric first network"
  echo "      - 'restart' - restart the fabric first network"
  echo "  ffn.sh -h (print this message)"
}

function prepare() {
  if [ ! -d "${FABRIC_SAMPLE_PATH}"/bin ]; then
    print_blue "===> Download the necessary dependencies"
    curl -sSL https://raw.githubusercontent.com/hyperledger/fabric/master/scripts/bootstrap.sh | bash -s -- 1.4.3 1.4.3 0.4.18
  fi
}

function networkUp() {
  prepare
  cd "${FABRIC_SAMPLE_PATH}"/first-network
  ./byfn.sh generate
  ./byfn.sh up -n
  cp -rf "${FABRIC_SAMPLE_PATH}"/first-network/crypto-config "${CURRENT_PATH}"
}

function networkDown() {
  prepare
  cd "${FABRIC_SAMPLE_PATH}"/first-network
  ./byfn.sh down
}

function networkRestart() {
  prepare
  cd "${FABRIC_SAMPLE_PATH}"/first-network
  ./byfn.sh restart -n
}

print_blue "===> Script version: $VERSION"

MODE=$1

if [ "$MODE" == "up" ]; then
  networkUp
elif [ "$MODE" == "down" ]; then
  networkDown
elif [ "$MODE" == "restart" ]; then
  networkRestart
else
  printHelp
  exit 1
fi
