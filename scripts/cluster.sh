#!/usr/bin/env bash

set -e
source x.sh

CURRENT_PATH=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
N=4

function GetPMName() {
  PM=''
  if [ "$(uname)" == "Darwin" ]; then
    DISTRO='MacOS'
    PM='brew'
  elif grep -Eqii "CentOS" /etc/issue || grep -Eq "CentOS" /etc/*-release; then
    DISTRO='CentOS'
    PM='yum'
  elif grep -Eqi "Red Hat Enterprise Linux Server" /etc/issue || grep -Eq "Red Hat Enterprise Linux Server" /etc/*-release; then
    DISTRO='RHEL'
    PM='yum'
  elif grep -Eqi "Aliyun" /etc/issue || grep -Eq "Aliyun" /etc/*-release; then
    DISTRO='Aliyun'
    PM='yum'
  elif grep -Eqi "Fedora" /etc/issue || grep -Eq "Fedora" /etc/*-release; then
    DISTRO='Fedora'
    PM='yum'
  elif grep -Eqi "Debian" /etc/issue || grep -Eq "Debian" /etc/*-release; then
    DISTRO='Debian'
    PM='apt-get'
  elif grep -Eqi "Ubuntu" /etc/issue || grep -Eq "Ubuntu" /etc/*-release; then
    DISTRO='Ubuntu'
    PM='apt-get'
  elif grep -Eqi "Raspbian" /etc/issue || grep -Eq "Raspbian" /etc/*-release; then
    DISTRO='Raspbian'
    PM='apt-get'
  else
    DISTRO='unknow'
  fi
  print_blue "Your OS distribution is detected as: "$DISTRO
  eval "$1=$PM"
}

function install_tmux(){
    if ! type tmux >/dev/null 2>&1; then
      print_blue "===> Install tmux with package manager"
      PM_NAME=''
      GetPMName PM_NAME
      if [ -n "$PM_NAME" ]; then
        if [ "$PM_NAME" == "brew" ]; then
          $PM_NAME install tmux
        else
          sudo $PM_NAME install -y tmux
        fi
      fi
    fi
}

function prepare() {
  print_blue "===> Generating $N nodes configuration"
  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"
  for ((i = 1; i < N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    mkdir ${root}
    cp -rf ${CURRENT_PATH}/package/* ${root}/
    cp -f ${PROJECT_PATH}/bin/axiom-ledger ${root}/tools/bin/
    echo "export AXIOM_LEDGER_PORT_JSONRPC=888${i}" >> ${root}/.env.sh
    echo "export AXIOM_LEDGER_PORT_WEBSOCKET=999${i}" >> ${root}/.env.sh
    echo "export AXIOM_LEDGER_PORT_P2P=400${i}" >> ${root}/.env.sh
    echo "export AXIOM_LEDGER_PORT_PPROF=5312${i}" >> ${root}/.env.sh
    echo "export AXIOM_LEDGER_PORT_MONITOR=4001${i}" >> ${root}/.env.sh
    ${root}/axiom-ledger config generate --default-node-index ${i}
  done
}

function splitWindow() {
  tmux splitw -v -p 50
  tmux splitw -h -p 50
  tmux selectp -t 0
  tmux splitw -h -p 50
}

function start_by_tmux() {
  print_blue "===> Staring cluster"
  tmux new -d -s axiom-ledger || (tmux kill-session -t axiom-ledger && tmux new -d -s axiom-ledger)

  for ((i = 0; i < N / 4; i = i + 1)); do
    splitWindow
    tmux new-window
  done
  splitWindow
  for ((i = 0; i < N; i = i + 1)); do
    tmux selectw -t $(($i / 4))
    tmux selectp -t $(($i % 4))
    tmux send-keys "${BUILD_PATH}/node$(($i + 1))/axiom-ledger start" C-m
  done
  tmux selectw -t 0
  tmux attach-session -t axiom-ledger
}

function start_by_nohup() {
  print_blue "===> Staring cluster"
  for ((i = 0; i < N; i = i + 1)); do
    ${BUILD_PATH}/node$(($i + 1))/start.sh
  done
}

if [ "$1" = "background" ]; then
  prepare
  start_by_nohup
else
  install_tmux
  prepare
  start_by_tmux
fi


