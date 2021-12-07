#!/usr/bin/env bash

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

BITXHUBBIN=$(which bitxhub)
MODE=$1
NODE=$2

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

function printHelp() {
  print_blue "Usage:  "
  echo "  vote.sh <mode> <number>"
  echo "    <mode> - delNode or addNode"
  echo "    <number> - node number"
  echo "  vote.sh -h (print this message)"
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
function addnode() {
  # 1. register addnode
  account=$(cat build/node${NODE}/account${NODE}.txt)
  pid=$(cat build/node${NODE}/pid${NODE}.txt)
  echo $account
  echo $pid

  # 1. proposal
  ${BITXHUBBIN} --repo=build/node1 client governance node register --account $account --type vpNode --pid $pid --id $NODE > build/node${NODE}/proposal.txt
  proposal=$(cat build/node${NODE}/proposal.txt | awk '{print $4}')
  sleep 5s
  #2. vote

  for i in {1..3} ; do
    ${BITXHUBBIN} --repo=build/node$i client governance vote --id $proposal --info approve --reason 1
  done
}

function delNode() {
  account=`${BITXHUBBIN} key address --path build/node$NODE/key.json`
  ${BITXHUBBINPATH}/bitxhub --repo=build/node4 client governance node logout --account $account --reason out node$NODE > node${NODE}/proposal.txt
  proposal=$(cat node${NODE}/proposal.txt | awk '{print $4}')
  sleep 5s
  #2. vote
  for i in {1..3} ; do
    ${BITXHUBBINPATH}/bitxhub --repo=build/node$i client governance vote --id $proposal --info approve --reason 1
  done
}

if [ $MODE == 'delNode' ]; then
  delNode
elif [ $MODE == 'addNode' ]; then
  addnode
else
  printHelp
fi
