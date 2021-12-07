#!/usr/bin/env bash
set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

CONSENSUSTYPE=$1
NUM=$2
BITXHUBBIN=$(which bitxhub)
AGENCYPRIVPATH=build/node$NUM/certs

SYSTEM=$(uname -s)
if [ $SYSTEM == "Linux" ]; then
  SYSTEM="linux"
elif [ $SYSTEM == "Darwin" ]; then
  SYSTEM="darwin"
fi
function print_green() {
  printf "${GREEN}%s${NC}\n" "$1"
}

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
}

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

# help prompt message
function printHelp() {
  print_blue "Usage:  "
  echo "  addNode.sh <consensus_type> <node_id>"
  echo "  'consensus_type' - bitxhub consensus type"
  echo "  'node_id' - node id which you want add or delete"
  echo "  addNode.sh -h (print this message)"
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


function generateNodeConfig() {
#
#
  rm -rf build/node$NUM
  print_blue "【1】generate configuration files"
  ${BITXHUBBIN} --repo build/node$NUM init
  print_blue "======> Generate configuration files for node $NUM"
  print_blue "【2】generate certs"
  ${BITXHUBBIN} cert priv gen --name node --target build/node$NUM/certs
  ${BITXHUBBIN} cert csr --key build/node$NUM/certs/node.priv --org node --target build/node$NUM/certs
  ${BITXHUBBIN} cert issue --csr build/node$NUM/certs/node.csr --is_ca false --key ${AGENCYPRIVPATH}/agency.priv --cert ${AGENCYPRIVPATH}/agency.cert --target build/node$NUM/certs
  rm build/node$NUM/certs/node.csr
  cp build/node1/certs/ca.cert build/node$NUM/certs

  print_blue "【3】generate key.json"
  ${BITXHUBBIN} key gen --target build/node$NUM --passwd bitxhub
#

#
  # Obtain node pid and address
  for (( i = 1; i <= $NUM; i++ )); do
    PID=`${BITXHUBBIN} cert priv pid --path build/node$i/certs/node.priv`
    pid_array+=(${PID})
    ADDR=`${BITXHUBBIN} key address --path build/node$i/key.json --passwd bitxhub`
    addr_array+=(${ADDR})
  done
}


# $1 : node number
# $2 : node startup repo
function rewriteNodeConfig() {
  print_blue "======> Rewrite config for node $NUM"

  print_blue "【1】rewrite bitxhub.toml"
  # port
  jsonrpc=888$NUM
  x_replace "s/jsonrpc.*= .*/jsonrpc = $jsonrpc/" build/node$NUM/bitxhub.toml
  grpc=6001$NUM
  x_replace "s/grpc.*= .*/grpc = $grpc/" build/node$NUM/bitxhub.toml
  gateway=909$NUM
  x_replace "s/gateway = .*/gateway = $gateway/" build/node$NUM/bitxhub.toml
  pprof=5312$NUM
  x_replace "s/pprof.*= .*/pprof = $pprof/" build/node$NUM/bitxhub.toml
  monitor=4001$NUM
  x_replace "s/monitor.*= .*/monitor = $monitor/" build/node$NUM/bitxhub.toml
  # mode
  if [ $CONSENSUSTYPE == "solo" ]; then
    x_replace "s/solo.*= .*/solo = true/" build/node$NUM/bitxhub.toml
  else
    x_replace "s/solo.*= .*/solo = false/" build/node$NUM/bitxhub.toml
  fi
  # order
  order_line=`sed -n '/\[order\]/=' build/node$NUM/bitxhub.toml | head -n 1`
  order_line=`expr $order_line + 1`
  x_replace "$order_line s/type.*= .*/type = \"$CONSENSUSTYPE\"/" build/node$NUM/bitxhub.toml

  print_blue "【2】rewrite network.toml"
  x_replace "1 s/id = .*/id = $NUM/" build/node$NUM/network.toml #要求第一行配置是自己的id
#  x_replace "s/n = .*/n = $NUM/" build/node$NUM/network.toml
  # nodes
#  if [ $NUM -gt 4 ]; then
  nodes_start=`sed -n '/\[\[nodes\]\]/=' build/node$NUM/network.toml | head -n 1`
  for (( i = 4; i < $NUM; i++ )); do
    x_replace "$nodes_start i\\
pid = \" \"
" build/node$NUM/network.toml
    x_replace "$nodes_start i\\
id = 1
" build/node$NUM/network.toml
    x_replace "$nodes_start i\\
hosts = [\"\/\ip4\/127.0.0.1\/tcp\/4001\/p2p\/\"]
" build/node$NUM/network.toml
    x_replace "$nodes_start i\\
account = \" \"
" build/node$NUM/network.toml
    x_replace "$nodes_start i\\
[[nodes]]
" build/node$NUM/network.toml
  done
#  fi
  for (( i = 1; i <= $NUM; i++ )); do
    account=${addr_array[$i-1]}
    ip=127.0.0.1
    pid=${pid_array[$i-1]}

    # 要求配置项顺序一定
    a_line=`sed -n "/account = \".*\"/=" build/node$NUM/network.toml | head -n $i | tail -n 1`
    x_replace "$a_line s/account = \".*\"/account = \"$account\"/" build/node$NUM/network.toml
    host_line=`expr $a_line + 1`
    x_replace "$host_line s/hosts = .*/hosts = [\"\/\ip4\/$ip\/tcp\/400$i\/p2p\/\"]/" build/node$NUM/network.toml
    id_line=`expr $a_line + 2`
    x_replace "$id_line s/id = .*/id = $i/" build/node$NUM/network.toml
    pid_line=`expr $a_line + 3`
    x_replace "$pid_line s/pid = \".*\"/pid = \"$pid\"/" build/node$NUM/network.toml
  done
  if [ $NUM -gt 4 ]; then
    x_replace "s/new.*= .*/new = true/" build/node$NUM/network.toml
  fi
  account=${addr_array[$NUM-1]}
  pid=${pid_array[$NUM-1]}
  echo $account > build/node$NUM/account${NUM}.txt
  echo $pid > build/node$NUM/pid${NUM}.txt
}

if [[ ! $1 || ! $2 ]]; then
  printHelp
  exit 1
fi

generateNodeConfig
rewriteNodeConfig