#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
CONFIG_PATH=${PROJECT_PATH}/config
BUILD_PATH=${CURRENT_PATH}/build
N=4


function Get_PM_Name() {
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

function prepare() {
  if ! type tmux >/dev/null 2>&1; then
    print_blue "===> Install tmux with package manager"
    PM_NAME=''
    Get_PM_Name PM_NAME
    if [ -n "$PM_NAME" ]; then
      if [ "$PM_NAME" == "brew" ]; then
        $PM_NAME install tmux
      else
        sudo $PM_NAME install -y tmux
      fi
    fi
  fi

  print_blue "===> Generating $N nodes configuration"
  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"
  for ((i = 1; i < N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}

    axiom --repo="${root}" config generate --default-node-index ${i}

    echo " #!/usr/bin/env bash" >"${root}"/start.sh
    echo "./axiom --repo \$(pwd)" start >>"${root}"/start.sh

    axiomConfig=${root}/axiom.toml
    networkConfig=${root}/network.toml
    x_replace "s/8881/888${i}/g" "${axiomConfig}"
    x_replace "s/9991/909${i}/g" "${axiomConfig}"
    x_replace "s/53121/5312${i}/g" "${axiomConfig}"
    x_replace "s/40011/4001${i}/g" "${axiomConfig}"
    x_replace "1s/1/${i}/" "${networkConfig}"
  done
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
  tmux new -d -s axiom || (tmux kill-session -t axiom && tmux new -d -s axiom)

  for ((i = 0; i < N / 4; i = i + 1)); do
    splitWindow
    tmux new-window
  done
  splitWindow
  for ((i = 0; i < N; i = i + 1)); do
    tmux selectw -t $(($i / 4))
    tmux selectp -t $(($i % 4))
    tmux send-keys "axiom --repo=${BUILD_PATH}/node$(($i + 1)) start" C-m
  done
  tmux selectw -t 0
  tmux attach-session -t axiom
}

function clear_config() {
  for ((i = 1; i < N + 1; i = i + 1)); do
    rm -rf ~/axiom${i}
  done
}

prepare
start
