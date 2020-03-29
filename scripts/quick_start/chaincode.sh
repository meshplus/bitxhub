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
  echo "  chaincode.sh <mode>"
  echo "    <mode> - one of 'up', 'down', 'restart'"
  echo "      - 'install <fabric_ip>' - install broker, transfer and data_swapper chaincode"
  echo "      - 'upgrade <fabric_ip> <chaincode_version(default: v1)>' - upgrade broker, transfer and data_swapper chaincode"
  echo "      - 'init <fabric_ip>' - init broker"
  echo "      - 'get_balance <fabric_ip>' - get Alice balance from transfer chaincode"
  echo "      - 'get_data <fabric_ip>' - get path value from data_swapper chaincode"
  echo "      - 'interchain_transfer <fabric_ip> <target_appchain_id>' - interchain transfer"
  echo "      - 'interchain_get <fabric_ip> <target_appchain_id>' - interchain get data"
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
  rm -rf "${CURRENT_PATH}"/config.yaml
  cp "${CURRENT_PATH}"/config-template.yaml "${CURRENT_PATH}"/config.yaml

  if [ ! -d crypto-config ]; then
    print_red "===> Please provide the 'crypto-config'"
    exit 1
  fi
}

function installChaincode() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  print_blue "===> Install chaincode at $FABRIC_IP"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  print_blue "===> 1. Deploying broker, transfer and data_swapper chaincode"
  fabric-cli chaincode install --gopath ./contracts --ccp broker --ccid broker --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode instantiate --ccp broker --ccid broker --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode install --gopath ./contracts --ccp transfer --ccid transfer --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode instantiate --ccp transfer --ccid transfer --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode install --gopath ./contracts --ccp data_swapper --ccid data_swapper --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode instantiate --ccp data_swapper --ccid data_swapper --config ./config.yaml --orgid org2 --user Admin --cid mychannel

  print_blue "===> 2. Set Alice 10000 amout in transfer chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"setBalance","Args":["Alice", "10000"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml

  print_blue "===> 3. Set (key: path, value: ${CURRENT_PATH}) in data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"set","Args":["path", "'"${CURRENT_PATH}"'"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml

  print_blue "===> 4. Register transfer and data_swapper chaincode to broker chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config ./config.yaml
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config ./config.yaml

  print_blue "===> 6. Audit transfer and data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "transfer", "1"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "data_swapper", "1"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml

}

function upgradeChaincode() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  CHAINCODE_VERSION=v1
  if [ $2 ]; then
    CHAINCODE_VERSION=$2
  fi

  print_blue "Upgrade chaincode at $FABRIC_IP"
  print_blue "Upgrade to version: $CHAINCODE_VERSION"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  print_blue "===> 1. Deploying broker, transfer and data_swapper chaincode"
  fabric-cli chaincode install --gopath ./contracts --ccp broker --ccid broker \
    --v $CHAINCODE_VERSION \
    --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode upgrade --ccp broker --ccid broker \
    --v $CHAINCODE_VERSION \
    --config ./config.yaml --orgid org2 --user Admin --cid mychannel

  fabric-cli chaincode install --gopath ./contracts --ccp transfer --ccid transfer \
    --v $CHAINCODE_VERSION \
    --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode upgrade --ccp transfer --ccid transfer \
    --v $CHAINCODE_VERSION \
    --config ./config.yaml --orgid org2 --user Admin --cid mychannel

  fabric-cli chaincode install --gopath ./contracts --ccp data_swapper --ccid data_swapper \
    --v $CHAINCODE_VERSION \
    --config ./config.yaml --orgid org2 --user Admin --cid mychannel
  fabric-cli chaincode upgrade --ccp data_swapper --ccid data_swapper \
    --v $CHAINCODE_VERSION \
    --config ./config.yaml --orgid org2 --user Admin --cid mychannel

  print_blue "===> 2. Set Alice 10000 amout in transfer chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"setBalance","Args":["Alice", "10000"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml

  print_blue "===> 3. Set (key: path, value: ${CURRENT_PATH}) in data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"set","Args":["path", "'"${CURRENT_PATH}"'"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml

  print_blue "===> 4. Register transfer and data_swapper chaincode to broker chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=transfer \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config ./config.yaml
  fabric-cli chaincode invoke --cid mychannel --ccid=data_swapper \
    --args='{"Func":"register"}' --user Admin --orgid org2 --payload --config ./config.yaml

  print_blue "===> 6. Audit transfer and data_swapper chaincode"
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "transfer", "1"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml
  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "data_swapper", "1"]}' \
    --user Admin --orgid org2 --payload --config ./config.yaml
}

function initBroker() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  print_blue "===> Init broker chaincode at $FABRIC_IP"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"initialize"}' \
    --user Admin --orgid org2 --payload --config ./config.yaml
}

function getBalance() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  print_blue "===> Query Alice balance at $FABRIC_IP"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  fabric-cli chaincode invoke --ccid=transfer \
    --args '{"Func":"getBalance","Args":["Alice"]}' \
    --config ./config.yaml --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

function getData() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  print_blue "===> Query data at $FABRIC_IP"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  fabric-cli chaincode invoke --ccid=data_swapper \
    --args '{"Func":"get","Args":["path"]}' \
    --config ./config.yaml --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

function interchainTransfer() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  if [ ! $2 ]; then
    echo "Please input target appchain"
    exit 1
  fi

  TARGET_APPCHAIN_ID=$2

  print_blue "===> Invoke at $FABRIC_IP"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  echo "===> Alice transfer token from one chain to another chain"
  echo "===> Target appchain id: $TARGET_APPCHAIN_ID"

  fabric-cli chaincode invoke --ccid transfer \
    --args '{"Func":"transfer","Args":["'"${TARGET_APPCHAIN_ID}"'", "mychannel&transfer", "Alice","Alice","1"]}' \
    --config ./config.yaml --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

function interchainGet() {
  prepare

  FABRIC_IP=localhost
  if [ $1 ]; then
    FABRIC_IP=$1
  fi

  if [ ! $2 ]; then
    echo "Please input target appchain"
    exit 1
  fi

  TARGET_APPCHAIN_ID=$2

  print_blue "===> Invoke at $FABRIC_IP"

  cd "${CURRENT_PATH}"
  export CONFIG_PATH=${CURRENT_PATH}
  x_replace "s/localhost/${FABRIC_IP}/g" config.yaml

  echo "===> Get path value from other appchain"
  echo "===> Target appchain id: $TARGET_APPCHAIN_ID"

  fabric-cli chaincode invoke --ccid data_swapper \
    --args '{"Func":"get","Args":["'"${TARGET_APPCHAIN_ID}"'", "mychannel&data_swapper", "path"]}' \
    --config ./config.yaml --payload \
    --orgid=org2 --user=Admin --cid=mychannel
}

MODE=$1

if [ "$MODE" == "install" ]; then
  shift
  installChaincode $1
elif [ "$MODE" == "upgrade" ]; then
  shift
  upgradeChaincode $1 $2
elif [ "$MODE" == "init" ]; then
  shift
  initBroker $1
elif [ "$MODE" == "get_balance" ]; then
  shift
  getBalance $1
elif [ "$MODE" == "get_data" ]; then
  shift
  getData $1
elif [ "$MODE" == "interchain_transfer" ]; then
  shift
  interchainTransfer $1 $2
elif [ "$MODE" == "interchain_get" ]; then
  shift
  interchainGet $1 $2
else
  printHelp
  exit 1
fi
