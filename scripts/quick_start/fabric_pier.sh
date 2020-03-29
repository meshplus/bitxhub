#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PIER_ROOT=${CURRENT_PATH}/.pier
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

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
}

function printHelp() {
  print_blue "Usage:  "
  echo "  fabric_pier.sh <mode>"
  echo "    <mode> - one of 'start', 'restart'"
  echo "      - 'start <bitxhub_addr> <fabric_ip> <pprof_port>' - bring up the fabric pier"
  echo "      - 'restart <bitxhub_addr> <fabric_ip> <pprof_port>' - restart the fabric pier"
  echo "      - 'id' - print pier id"
  echo "  fabric_pier.sh -h (print this message)"
}

function prepare() {
  cd "${CURRENT_PATH}"
  if [ ! -d pier ]; then
    print_blue "===> Clone pier"
    git clone git@git.hyperchain.cn:dmlab/pier.git
  fi

  print_blue "===> Compile pier"
  cd pier
  make install

  cd "${CURRENT_PATH}"
  if [ ! -d pier-client-fabric ]; then
    print_blue "===> Clone pier-client-fabric"
    git clone git@git.hyperchain.cn:dmlab/pier-client-fabric.git
  fi

  print_blue "===> Compile pier-client-fabric"
  cd pier-client-fabric
  make fabric1.4

  cd "${CURRENT_PATH}"
  if [ ! -d crypto-config ]; then
    print_red "===> Please provide the 'crypto-config'"
    exit 1
  fi

  if [ ! -f config-template.yaml ]; then
    print_blue "===> Download config-template.yaml"
    wget https://raw.githubusercontent.com/meshplus/bitxhub/master/scripts/quick_start/config-template.yaml
  fi
  rm -rf "${CURRENT_PATH}"/config.yaml
  cp "${CURRENT_PATH}"/config-template.yaml "${CURRENT_PATH}"/config.yaml

  if [ ! -f fabric-rule.wasm ]; then
    print_blue "===> Download fabric-rule.wasm"
    wget https://raw.githubusercontent.com/meshplus/bitxhub/master/scripts/quick_start/fabric-rule.wasm
  fi
}

function start() {
  prepare

  pier --repo="${PIER_ROOT}" init
  mkdir -p "${PIER_ROOT}"/plugins
  cp "${CURRENT_PATH}"/pier-client-fabric/build/fabric-client-1.4.so "${PIER_ROOT}"/plugins/
  cp -rf "${CURRENT_PATH}"/pier-client-fabric/config "${PIER_ROOT}"/fabric
  cp -rf "${CURRENT_PATH}"/crypto-config "${PIER_ROOT}"/fabric/
  cp -rf "${CURRENT_PATH}"/config.yaml "${PIER_ROOT}"/fabric/
  cp "${PIER_ROOT}"/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem \
    "${PIER_ROOT}"/fabric/fabric.validators

  BITXHUB_ADDR="localhost:60011"
  FABRIC_IP=localhost
  PPROF_PORT=44555

  if [ $1 ]; then
    BITXHUB_ADDR=$1
  fi

  if [ $2 ]; then
    FABRIC_IP=$2
  fi

  if [ $3 ]; then
    PPROF_PORT=$3
  fi

  x_replace "s/44555/${PPROF_PORT}/g" "${PIER_ROOT}"/pier.toml
  x_replace "s/localhost:60011/${BITXHUB_ADDR}/g" "${PIER_ROOT}"/pier.toml
  x_replace "s/localhost/${FABRIC_IP}/g" "${PIER_ROOT}"/fabric/fabric.toml

  print_blue "===> bitxhub_addr: $BITXHUB_ADDR, fabric_ip: $FABRIC_IP, pprof: $PPROF_PORT"

  print_blue "===> Register pier to bitxhub"
  pier --repo "${PIER_ROOT}" appchain register \
    --name chainA \
    --type fabric \
    --desc chainA-description \
    --version 1.4.3 \
    --validators "${PIER_ROOT}"/fabric/fabric.validators

  print_blue "===> Deploy rule in bitxhub"
  pier --repo "${PIER_ROOT}" rule deploy --path "${CURRENT_PATH}"/fabric-rule.wasm

  print_blue "===> Start pier"
  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${PIER_ROOT}/fabric
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  pier --repo "${PIER_ROOT}" start
}

function restart() {
  prepare

  BITXHUB_ADDR="localhost:60011"
  FABRIC_IP=localhost
  PPROF_PORT=44555

  if [ $1 ]; then
    BITXHUB_ADDR=$1
  fi

  if [ $2 ]; then
    FABRIC_IP=$2
  fi

  if [ $3 ]; then
    PPROF_PORT=$3
  fi

  print_blue "===> bitxhub_addr: $BITXHUB_ADDR, fabric_ip: $FABRIC_IP, pprof: $PPROF_PORT"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${PIER_ROOT}/fabric
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml
  pier --repo "${PIER_ROOT}" start
}

function printID() {
  if [ ! -d $PIER_ROOT ]; then
    print_red "Please start pier firstly"
    exit 1
  fi

  pier --repo "${PIER_ROOT}" id
}

MODE=$1

if [ "$MODE" == "start" ]; then
  shift
  start $1 $2 $3
elif [ "$MODE" == "restart" ]; then
  shift
  restart $1 $2 $3
elif [ "$MODE" == "id" ]; then
  printID
else
  printHelp
  exit 1
fi
