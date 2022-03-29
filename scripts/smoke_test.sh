set -e
source x.sh


CURRENT_PATH=$(pwd)
function printHelp() {
    print_blue "Usage:  "
    echo "  bxh_test.sh [-b <BRANCH_NAME>] [-t <TEST_NAME>]"
    echo "  -'b' - the branch of base ref"
    echo "  -'t' - the test name such as bitxhub-tester,gosdk-tester,http-tester"
    echo "  bxh_test.sh -h (print this message)"
}

function Get_PM_Name(){
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
  eval "$2=$DISTRO"
}

function prepare() {
    print_blue "===> 1. Install packr"
    if ! type packr >/dev/null 2>&1; then
      go get -u github.com/gobuffalo/packr/packr
    fi
    print_blue "===> 2. Install tmux with package manager"
    if ! type tmux >/dev/null 2>&1; then
      PM_NAME=''
      Get_PM_Name PM_NAME
      if [ -n "$PM_NAME" ]; then
        if [ "$PM_NAME" == "brew" ]; then
          $PM_NAME install tmux
        else
          sudo "$PM_NAME" install -y tmux
        fi
      fi
    fi
}

function start_bxh_solo() {
    print_blue "Start bitxhub"
    echo "$CURRENT_PATH"
    cd ../ && make install && cd scripts
    print_blue "Start Solo"
    nohup bash solo.sh 2>gc.log 1>solo.log &
    sleep 100
}

function bitxhub_tester() {
    print_blue "Start git clone Premo"
    echo "$BRANCH_NAME"
    cd ../ && git clone -b "$BRANCH_NAME" https://github.com/meshplus/premo.git
    cd premo && make install && premo init
    print_blue "Start test"
    make smoke-tester
}

function bxh_test() {
    prepare
    start_bxh_solo
    bitxhub_tester
}

while getopts "h?b:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  b)
    BRANCH_NAME=$OPTARG
    ;;
  esac
done

bxh_test