#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
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
  echo "  chaincode.sh <mode> [-c <config_path>] [-v <chaincode_version>] [-t <target_appchain_id>]"
  echo "    <mode> - one of 'install', 'upgrade', 'init','get_balance','get_data','interchain_transfer','interchain_get'"
  echo "      - 'install' - install broker, transfer and data_swapper chaincode"
  echo "      - 'upgrade <chaincode_version(default: v1)>' - upgrade broker, transfer and data_swapper chaincode"
  echo "      - 'init' - init broker"
  echo "      - 'get_balance' - get Alice balance from transfer chaincode"
  echo "      - 'get_data' - get path value from data_swapper chaincode"
  echo "      - 'interchain_transfer' - interchain transfer"
  echo "      - 'interchain_get' - interchain get data"
  echo "    -c <config_path> - specify which config.yaml file use (default \"./config.yaml\")"
  echo "    -v <chaincode_version> - upgrade fabric chaincode version (default \"v1\")"
  echo "    -t <target_appchain_id> - when inter-chain interaction is required"
  echo "  chaincode.sh -h (print this message)"
}

function prepare() {
  if ! type fabric-cli >/dev/null 2>&1; then
    print_blue "===> Install fabric-cli"
    go get github.com/securekey/fabric-examples/fabric-cli/cmd/fabric-cli
  fi

  if [ ! -d contracts ]; then
    print_blue "===> Download chaincode"
    wget https://github.com/meshplus/bitxhub/raw/master/scripts/quick_start/contracts.zip
    unzip -q contracts.zip
    rm contracts.zip
  fi

  if [ ! -f config-template.yaml ]; then
    print_blue "===> Download config-template.yaml"
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


  if [ ! -d crypto-config ]; then
    print_red "===> Please provide the 'crypto-config'(first fabric network)"
    exit 1
  fi

  if [ ! -d crypto-configB ]; then
    print_red "===> Please provide the 'crypto-configB'(second fabric network)"
    exit 1
  fi
}

function installChaincode() {
  prepare

  print_blue "===> Install chaincode"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}

  print_blue "===> 1. Deploying broker, transfer and data_swapper chaincode"
  fabric-cli chaincode install --gopath ./contracts --ccp broker --ccid broker --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode instantiate --ccp broker --ccid broker --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode install --gopath ./contracts --ccp transfer --ccid transfer --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode instantiate --ccp transfer --ccid transfer --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode install --gopath ./contracts --ccp data_swapper --ccid data_swapper --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode instantiate --ccp data_swapper --ccid data_swapper --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

  print_blue "===> 2. Set Alice 10000 amout in transfer chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"setBalance","Args":["Alice", "10000"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

  print_blue "===> 3. Set (key: path, value: ${CURRENT_PATH}) in data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"set","Args":["path", "'"${CURRENT_PATH}"'"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

  print_blue "===> 4. Register transfer and data_swapper chaincode to broker chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

  print_blue "===> 6. Audit transfer and data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "transfer", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "data_swapper", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

}

function upgradeChaincode() {
  prepare

  print_blue "Upgrade to version: $CHAINCODE_VERSION"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}


  print_blue "===> 1. Deploying broker, transfer and data_swapper chaincode"
  fabric-cli chaincode install --gopath ./contracts --ccp broker --ccid broker \
    --v $CHAINCODE_VERSION \
    --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode upgrade --ccp broker --ccid broker \
    --v $CHAINCODE_VERSION \
    --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

  fabric-cli chaincode install --gopath ./contracts --ccp transfer --ccid transfer \
    --v $CHAINCODE_VERSION \
    --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode upgrade --ccp transfer --ccid transfer \
    --v $CHAINCODE_VERSION \
    --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

  fabric-cli chaincode install --gopath ./contracts --ccp data_swapper --ccid data_swapper \
    --v $CHAINCODE_VERSION \
    --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode upgrade --ccp data_swapper --ccid data_swapper \
    --v $CHAINCODE_VERSION \
    --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

  print_blue "===> 2. Set Alice 10000 amout in transfer chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"setBalance","Args":["Alice", "10000"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

  print_blue "===> 3. Set (key: path, value: ${CURRENT_PATH}) in data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"set","Args":["path", "'"${CURRENT_PATH}"'"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

  print_blue "===> 4. Register transfer and data_swapper chaincode to broker chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"

  print_blue "===> 6. Audit transfer and data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "transfer", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "data_swapper", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
}

function initBroker() {
  prepare

  print_blue "===> Init broker chaincode"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}


  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"initialize"}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
}

function getBalance() {
  prepare

  print_blue "===> Query Alice balance"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}


  fabric-cli chaincode invoke --ccid=transfer \
    --args '{"Func":"getBalance","Args":["Alice"]}' \
    --config "${CONFIG_YAML}" --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

function getData() {
  prepare

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}


  fabric-cli chaincode invoke --ccid=data_swapper \
    --args '{"Func":"get","Args":["path"]}' \
    --config "${CONFIG_YAML}" --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

function interchainTransfer() {
  prepare

  if [ ! $TARGET_APPCHAIN_ID ]; then
    echo "Please input target appchain"
    exit 1
  fi

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}


  echo "===> Alice transfer token from one chain to another chain"
  echo "===> Target appchain id: $TARGET_APPCHAIN_ID"

  fabric-cli chaincode invoke --ccid transfer \
    --args '{"Func":"transfer","Args":["'"${TARGET_APPCHAIN_ID}"'", "mychannel&transfer", "Alice","Alice","1"]}' \
    --config "${CONFIG_YAML}" --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

function interchainGet() {
  prepare

  if [ ! $TARGET_APPCHAIN_ID ]; then
    echo "Please input target appchain"
    exit 1
  fi

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}


  echo "===> Get path value from other appchain"
  echo "===> Target appchain id: $TARGET_APPCHAIN_ID"

  fabric-cli chaincode invoke --ccid data_swapper \
    --args '{"Func":"get","Args":["'"${TARGET_APPCHAIN_ID}"'", "mychannel&data_swapper", "path"]}' \
    --config "${CONFIG_YAML}" --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}


CONFIG_YAML=./config.yaml
CHAINCODE_VERSION=v1
TARGET_APPCHAIN_ID=""



MODE=$1
shift

while getopts "h?c:v:t:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  c)
    CONFIG_YAML=$OPTARG
    ;;
  v)
    CHAINCODE_VERSION=$OPTARG
    ;;
  t)
    TARGET_APPCHAIN_ID=$OPTARG
    ;;
  esac
done

if [ "$MODE" == "install" ]; then
  installChaincode
elif [ "$MODE" == "upgrade" ]; then
  upgradeChaincode
elif [ "$MODE" == "init" ]; then
  initBroker
elif [ "$MODE" == "get_balance" ]; then
  getBalance
elif [ "$MODE" == "get_data" ]; then
  getData
elif [ "$MODE" == "interchain_transfer" ]; then
  interchainTransfer
elif [ "$MODE" == "interchain_get" ]; then
  interchainGet
else
  printHelp
  exit 1
fi

