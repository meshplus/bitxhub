#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
CONFIG_PATH=${PROJECT_PATH}/config
BUILD_PATH=${CURRENT_PATH}/build
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'
N=4

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
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

function prepare() {
  print_blue "===> Generating $N nodes configuration"
  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"
  for ((i = 1; i < N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    mkdir -p "${root}"

    cp -rf "${CURRENT_PATH}"/certs/node${i}/certs "${root}"
    cp -rf "${CONFIG_PATH}"/* "${root}"

    echo " #!/usr/bin/env bash" >"${root}"/start.sh
    echo "./bitxhub --root \$(pwd)" start >>"${root}"/start.sh

    bitxhubConfig=${root}/bitxhub.toml
    networkConfig=${root}/network.toml
    x_replace "s/60011/6001${i}/g" "${bitxhubConfig}"
    x_replace "s/9091/909${i}/g" "${bitxhubConfig}"
    x_replace "s/53121/5312${i}/g" "${bitxhubConfig}"
    x_replace "s/40011/4001${i}/g" "${bitxhubConfig}"
    x_replace "s/8881/888${i}/g" "${bitxhubConfig}"
    x_replace "1s/1/${i}/" "${networkConfig}"
  done

  print_blue "===> Building plugin"
  cd "${PROJECT_PATH}"/internal/plugins
  make raft${TAGS}

  for ((i = 1; i < N + 1; i = i + 1)); do
    cp -rf "${PROJECT_PATH}"/internal/plugins/build "${BUILD_PATH}"/node${i}/plugins
  done
}

function compile() {
  print_blue "===> Compiling bitxhub"
  cd "${PROJECT_PATH}"
  make install${TAGS}
}

function splitWindow() {
  tmux splitw -v -p 50
  tmux splitw -h -p 50
  tmux selectp -t 0
  tmux splitw -h -p 50
}

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

function clear_config() {
  for ((i = 1; i < N + 1; i = i + 1)); do
    rm -rf ~/bitxhub${i}
  done
}

prepare
start
