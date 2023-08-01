set -e
source x.sh


CURRENT_PATH=$(pwd)
function printHelp() {
    print_blue "Usage:  "
    echo "  bxh_test.sh [-b <BRANCH_NAME>] [-t <TEST_NAME>]"
    echo "  -'b' - the branch of base ref"
    echo "  bxh_test.sh -h (print this message)"
}

function start_bxh_solo() {
    print_blue "===> 1. Start solo axiom"
    echo "$CURRENT_PATH"
    cd ../ && make install && cd scripts
    nohup bash solo.sh 2>gc.log 1>solo.log &
    sleep 10
}

function axiom_tester() {
    print_blue "===> 2. Start git clone snake"
    echo "$BRANCH_NAME"
    cd ../
    git clone -b "$BRANCH_NAME" https://github.com/axiomesh/snake.git
    cd snake
    print_blue "Start test"
    print_blue "===> 3. Start smoke test"
    npm install && npm run smoke-test
}

function bxh_test() {
    start_bxh_solo
    axiom_tester
}
BRANCH_NAME="main"
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