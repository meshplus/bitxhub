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
  if [ ! -d "${FABRIC_SAMPLE_PATH}"/second-network ]; then
    createSecondNetwork
  fi
  docker volume prune -f
}

function createSecondNetwork() {
  cd "${FABRIC_SAMPLE_PATH}"
  cp -r first-network second-network
  cd "${FABRIC_SAMPLE_PATH}"/second-network
  x_replace 's/7050:/7055:/g' base/docker-compose-base.yaml
  x_replace 's/7051:/7052:/g' base/docker-compose-base.yaml
  x_replace 's/8051:/8052:/g' base/docker-compose-base.yaml
  x_replace 's/9051:/9052:/g' base/docker-compose-base.yaml
  x_replace 's/10051:/10052:/g' base/docker-compose-base.yaml
  x_replace 's/example/example1/g' base/docker-compose-base.yaml

  #    x_replace 's/FABRIC_LOGGING_SPEC=INFO/FABRIC_LOGGING_SPEC=DEBUG/g' base/peer-base.yaml
  x_replace 's/_byfn/_byfn1/g' base/peer-base.yaml
  #    x_replace 's/CORE_PEER_TLS_ENABLED=true/CORE_PEER_TLS_ENABLED=false/g' base/peer-base.yaml
  #    x_replace 's/ORDERER_GENERAL_TLS_ENABLED=true/ORDERER_GENERAL_TLS_ENABLED=false/g' base/peer-base.yaml
  x_replace 's/cli/cli1/g' docker-compose-cli.yaml
  x_replace 's/byfn/byfn1/g' docker-compose-cli.yaml
  x_replace 's/example/example1/g' docker-compose-cli.yaml

  #    x_replace 's/CORE_PEER_TLS_ENABLED=true/CORE_PEER_TLS_ENABLED=false/g' docker-compose-cli.yaml

  x_replace 's/example/example1/g' byfn.sh
  x_replace 's/exec cli/exec cli1/g' byfn.sh

  x_replace 's/example/example1/g' crypto-config.yaml

  x_replace 's/7050/7055/g' ccp-generate.sh
  x_replace 's/7051/7052/g' ccp-generate.sh
  x_replace 's/8051/8052/g' ccp-generate.sh
  x_replace 's/9051/9052/g' ccp-generate.sh
  x_replace 's/10051/10052/g' ccp-generate.sh
  x_replace 's/example/example1/g' ccp-generate.sh

  x_replace 's/first-network/second-network/g' ccp-template.json
  x_replace 's/example/example1/g' ccp-template.json

  x_replace 's/example/example1/g' ccp-template.yaml
  x_replace 's/first-network/second-network/g' ccp-template.yaml

  x_replace 's/example/example1/g' configtx.yaml

  x_replace 's/.example.com/.example1.com/g' scripts/script.sh

  x_replace 's/example/example1/g' scripts/utils.sh

  #    x_replace 's/FABRIC_CA_SERVER_TLS_ENABLED=true/FABRIC_CA_SERVER_TLS_ENABLED=false/g' docker-compose-e2e-template.yaml
  x_replace 's/example/example1/g' docker-compose-e2e-template.yaml
  x_replace 's/byfn/byfn1/g' docker-compose-e2e-template.yaml

  x_replace 's/example/example1/g' docker-compose-org3.yaml
  #    x_replace 's/CORE_PEER_TLS_ENABLED=true/CORE_PEER_TLS_ENABLED=false/g' docker-compose-org3.yaml
}

function networkUp() {
  prepare

  cd "${FABRIC_SAMPLE_PATH}"/first-network
  ./byfn.sh generate
  ./byfn.sh up -n
  rm -rf "${CURRENT_PATH}"/crypto-config
  cp -rf "${FABRIC_SAMPLE_PATH}"/first-network/crypto-config "${CURRENT_PATH}"/crypto-config

  cd "${FABRIC_SAMPLE_PATH}"/second-network
  ./byfn.sh generate
  ./byfn.sh up -n
  rm -rf "${CURRENT_PATH}"/crypto-configB
  cp -rf "${FABRIC_SAMPLE_PATH}"/second-network/crypto-config "${CURRENT_PATH}"/crypto-configB
}

function networkDown() {
  prepare
  # stop all fabric nodes
  cd "${FABRIC_SAMPLE_PATH}"/first-network
  ./byfn.sh down
}

function networkRestart() {
  prepare

  cd "${FABRIC_SAMPLE_PATH}"/first-network
  ./byfn.sh restart -n
  cd "${FABRIC_SAMPLE_PATH}"/second-network
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
