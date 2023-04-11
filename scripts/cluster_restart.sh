#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
BUILD_PATH=${CURRENT_PATH}/build
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'
N=4

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
}

function splitWindow() {
  tmux splitw -v -p 50
  tmux splitw -h -p 50
  tmux selectp -t 0
  tmux splitw -h -p 50
}

if [ ! -d "${BUILD_PATH}" ]; then
  print_red "bitxhub's nodes dir is not exit, Please run make cluster first"
  exit 1
fi

function start() {
  print_blue "===> Staring cluster"
  #osascript ${PROJECT_PATH}/scripts/split.scpt ${N} ${DUMP_PATH}/cluster/node
  tmux new -d -s bitxhub || (tmux kill-session -t bitxhub && tmux new -d -s bitxhub)

  for ((i = 0; i < N / 4; i = i + 1)); do
    splitWindow
    tmux new-window
  done
  splitWindow
  for ((i = 0; i < N; i = i + 1)); do
    tmux selectw -t $(($i / 4))
    tmux selectp -t $(($i % 4))
    tmux send-keys "bitxhub --repo=${BUILD_PATH}/node$(($i + 1)) start" C-m
  done
  tmux selectw -t 0
  tmux attach-session -t bitxhub
}

start
