#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
N=$1

function prepare() {
  bash build.sh "${N}"
}

function compile() {
  cd "${PROJECT_PATH}"
  make install
}

function splitWindow() {
  tmux splitw -v -p 50
  tmux splitw -h -p 50
  tmux selectp -t 0
  tmux splitw -h -p 50
}

function start() {
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
    tmux send-keys "ulimit -n 20000" C-m
    tmux send-keys "bitxhub --repo=${BUILD_PATH}/node$(($i + 1)) start" C-m
  done
  tmux selectw -t 0
  tmux attach-session -t bitxhub
}

function clear_config() {
  for ((i = 1; i < N + 1; i = i + 1)); do
    rm -rf ~/bitxhub${i}
  done
}

prepare
compile
start
