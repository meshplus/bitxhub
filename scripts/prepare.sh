#!/usr/bin/env bash

BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

print_blue "===> 1. Install packr"
if ! type packr >/dev/null 2>&1; then
  go get -u github.com/gobuffalo/packr/packr
fi

print_blue "===> 2. Install golangci-lint"
if ! type golanci-lint >/dev/null 2>&1; then
  go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.23.0
fi

print_blue "===> 3. Install go mock tool"
if ! type gomock >/dev/null 2>&1; then
  go get github.com/golang/mock/gomock
fi
if ! type mockgen >/dev/null 2>&1; then
  go get github.com/golang/mock/mockgen
fi

function Get_PM_Name()
{
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
  print_blue "Your OS distribution is detected as: "$DISTRO;
  eval "$1=$PM"
}

print_blue "===> 4. Install tmux with package manager"
PM_NAME=''
Get_PM_Name PM_NAME
if [ -n "$PM_NAME" ]; then
  if [ "$PM_NAME" == "brew" ]; then
    $PM_NAME install tmux
  else
    sudo $PM_NAME install -y tmux
  fi
fi
