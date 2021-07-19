#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PIER_VERSION=v1.0.0-rc1
PIER_CLIENT_FABRIC_VERSION=v1.0.0-rc1
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
  echo "  fabric_pier.sh <mode> [-r <pier_root>] [-c <crypto_config_path>] [-g <config_path>] [-p <pier_port>] [-b <bitxhub_addr>] [-o <pprof_port>]"
  echo "    <mode> - one of 'start', 'restart', 'id'"
  echo "      - 'start - bring up the fabric pier"
  echo "      - 'restart' - restart the fabric pier"
  echo "      - 'id' - print pier id"
  echo "    -r <pier_root> - pier repo path (default \".pier\")"
  echo "    -c <crypto_config_path> - specify which crypto-config dir use (default \"./crypto-config\")"
  echo "    -g <config_path> - config path (default \"./config.yaml\")"
  echo "    -p <pier_port> - pier port (default \"8987\")"
  echo "    -b <bitxhub_addr> - bitxhub addr(default \"localhost:60011\")"
  echo "    -o <pprof_port> - pier pprof port(default \"44555\")"
  echo "  fabric_pier.sh -h (print this message)"
}

function prepare() {
  cd "${CURRENT_PATH}"
  if [ ! -d pier ]; then
    print_blue "===> Cloning meshplus/pier repo and checkout ${PIER_VERSION}"
    git clone https://github.com/meshplus/pier.git &&
      cd pier && git checkout ${PIER_VERSION}
  fi

  print_blue "===> Compiling meshplus/pier"
  cd "${CURRENT_PATH}"/pier
  make install

  cd "${CURRENT_PATH}"
  if [ ! -d pier-client-fabric ]; then
    print_blue "===> Cloning meshplus/pier-client-fabric repo and checkout ${PIER_CLIENT_FABRIC_VERSION}"
    git clone https://github.com/meshplus/pier-client-fabric.git &&
      cd pier-client-fabric && git checkout ${PIER_CLIENT_FABRIC_VERSION}
  fi

  print_blue "===> Compiling meshplus/pier-client-fabric"
  cd "${CURRENT_PATH}"/pier-client-fabric
  make fabric1.4

  cd "${CURRENT_PATH}"
  if [ ! -d crypto-config ]; then
    print_red "===> Please provide the 'crypto-config'(first fabric network)"
    exit 1
  fi

  if [ ! -d crypto-configB ]; then
    print_red "===> Please provide the 'crypto-configB'(second fabric network)"
    exit 1
  fi

  if [ ! -f config-template.yaml ]; then
    print_blue "===> Downloading config-template.yaml"
    wget https://raw.githubusercontent.com/meshplus/bitxhub/master/scripts/quick_start/config-template.yaml
  fi

  if [ ! -f config.yaml ]; then
    cp "${CURRENT_PATH}"/config-template.yaml "${CURRENT_PATH}"/config.yaml
  fi

  if [ ! -f configB.yaml ]; then
    cp "${CURRENT_PATH}"/config-template.yaml "${CURRENT_PATH}"/configB.yaml
    x_replace 's/7050/7055/g' "${CURRENT_PATH}"/configB.yaml
    x_replace 's/7051/7052/g' "${CURRENT_PATH}"/configB.yaml
    x_replace 's/8051/8052/g' "${CURRENT_PATH}"/configB.yaml
    x_replace 's/9051/9052/g' "${CURRENT_PATH}"/configB.yaml
    x_replace 's/10051/10052/g' "${CURRENT_PATH}"/configB.yaml
    x_replace 's/crypto-config/crypto-configB/g' "${CURRENT_PATH}"/configB.yaml
    x_replace 's/example/example1/g' "${CURRENT_PATH}"/configB.yaml
  fi

  if [ ! -f fabric_rule.wasm ]; then
    print_blue "===> Downloading fabric_rule.wasm"
    wget https://raw.githubusercontent.com/meshplus/bitxhub/master/scripts/quick_start/fabric_rule.wasm
  fi
}

function start() {
  prepare

  pier --repo="${PIER_ROOT}" init
  cp "${CURRENT_PATH}"/pier-client-fabric/build/fabric-client-1.4.so "${PIER_ROOT}"/plugins/
  cp -rf "${CURRENT_PATH}"/pier-client-fabric/config "${PIER_ROOT}"/fabric
  rm -rf "${PIER_ROOT}"/fabric/crypto-config
  cp -rf "${CRYPTO_CONFIG}" "${PIER_ROOT}"/fabric/crypto-config
  cp -rf "${CONFIG}" "${PIER_ROOT}"/fabric/config.yaml
  x_replace 's/crypto-configB/crypto-config/g' "${PIER_ROOT}"/fabric/config.yaml
  PEM_PATH="${PIER_ROOT}"/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem
  if [ ! -f "${PEM_PATH}" ]; then
    PEM_PATH="${PIER_ROOT}"/fabric/crypto-config/peerOrganizations/org2.example1.com/peers/peer1.org2.example1.com/msp/signcerts/peer1.org2.example1.com-cert.pem
  fi
  cp "${PEM_PATH}" "${PIER_ROOT}"/fabric/fabric.validators

  x_replace "s/8987/${PIER_PORT}/g" "${PIER_ROOT}"/pier.toml
  x_replace "s/44555/${PPROF_PORT}/g" "${PIER_ROOT}"/pier.toml
  x_replace "s/localhost:60011/${BITXHUB_ADDR}/g" "${PIER_ROOT}"/pier.toml

  print_blue "===> pier_root: $PIER_ROOT, pier_port: $PIER_PORT, bitxhub_addr: $BITXHUB_ADDR, pprof: $PPROF_PORT"

  print_blue "===> Register pier to bitxhub"
  pier --repo "${PIER_ROOT}" appchain register \
    --name chainA \
    --type fabric \
    --desc chainA-description \
    --version 1.4.3 \
    --validators "${PIER_ROOT}"/fabric/fabric.validators

  print_blue "===> Deploy rule in bitxhub"
  pier --repo "${PIER_ROOT}" rule deploy --path "${CURRENT_PATH}"/fabric_rule.wasm

  print_blue "===> Start pier"
  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${PIER_ROOT}/fabric

  pier --repo "${PIER_ROOT}" start
}

function restart() {
  prepare

  print_blue "===> pier_root: $PIER_ROOT,bitxhub_addr: $BITXHUB_ADDR, pprof: $PPROF_PORT"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${PIER_ROOT}/fabric
  pier --repo "${PIER_ROOT}" start
}

function printID() {
  if [ ! -d "$PIER_ROOT" ]; then
    print_red "Please start pier firstly"
    exit 1
  fi

  pier --repo "${PIER_ROOT}" id
}

PIER_ROOT=${CURRENT_PATH}/.pier
CRYPTO_CONFIG="${CURRENT_PATH}"/crypto-config
CONFIG="${CURRENT_PATH}"/config.yaml
PIER_PORT=8987
BITXHUB_ADDR="localhost:60011"
PPROF_PORT=44555

MODE=$1
shift

while getopts "h?r:c:g:p:b:o:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  r)
    PIER_ROOT=$OPTARG
    ;;
  c)
    CRYPTO_CONFIG=$OPTARG
    ;;
  g)
    CONFIG=$OPTARG
    ;;
  p)
    PIER_PORT=$OPTARG
    ;;
  b)
    BITXHUB_ADDR=$OPTARG
    ;;
  o)
    PPROF_PORT=$OPTARG
    ;;
  esac
done

if [ "$MODE" == "start" ]; then
  start
elif [ "$MODE" == "restart" ]; then
  restart
elif [ "$MODE" == "id" ]; then
  printID
else
  printHelp
  exit 1
fi
