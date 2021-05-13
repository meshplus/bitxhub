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
      sudo apt-get install -y tmux
    fi
}

function startBitxhub() {
    print_blue "Start bitxhub"
    echo "$CURRENT_PATH"
    cd ../ && make install && cd scripts
    print_blue "Start Solo"
    nohup bash solo.sh 2>gc.log 1>solo.log &
    while  lsof -i:60011 > /dev/null ;do
      sleep 10
    done
}
function bitxhub_tester() {
    print_blue "Start git clone Premo"
    echo "$BRANCH_NAME"
    cd ../ && git clone -b "$BRANCH_NAME" https://github.com/meshplus/premo.git
    cd premo && make install && premo init
    print_blue "Start test"
    make bitxhub-tester
}
function bxh_test() {
    prepare
    startBitxhub
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