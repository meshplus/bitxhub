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
function prepare() {
    print_blue "===> 1. Install packr"
    if ! type packr >/dev/null 2>&1; then
      go get -u github.com/gobuffalo/packr/packr
    fi
    print_blue "===> 2. Install tmux with package manager"
    if ! type tmux >/dev/null 2>&1; then
      sudo apt-get -y tmux
    fi
}

function startBitxhub() {
    print_blue "Start bitxhub"
    echo "$CURRENT_PATH"
    cd ../ && make cluster
}
function getPremo() {
  print_blue "Start git clone Premo"
  git clone -b "$BRANCH_NAME" https://github.com/meshplus/premo.git
}
function test() {
  print_blue "Start $TEST_NAME test"
  cd "$CURRENT_PATH"/premo && make "$TEST_NAME"
}
function bxh_test() {
    startBitxhub
}
while getopts "h?b:t" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  b)
    BRANCH_NAME=$OPTARG
    ;;
  t)
    TEST_NAME=$OPTARG
    ;;
  esac
done
bxh_test