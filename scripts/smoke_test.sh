set -e
source x.sh
source smoke_env.sh

CURRENT_PATH=$(pwd)
function printHelp() {
    print_blue "Usage:  "
    echo "  axiom_test.sh [-b <BRANCH_NAME>] [-t <TEST_NAME>]"
    echo "  -'b' - the branch of base ref"
    echo "  axiom_test.sh -h (print this message)"
}

function start_axiom_solo() {
    print_blue "===> 1. Start solo axiom"
    echo "$CURRENT_PATH"
    cd ../ && make install && cd scripts
    nohup bash solo.sh 2>gc.log 1>solo.log &
    sleep 10
}

function start_axiom_rbft() {
    print_blue "===> 1. Start rbft axiom"
    echo "$CURRENT_PATH"
    cd ../ && make install && cd scripts
    nohup bash cluster.sh 2>gc.log 1>cluster.log &
    sleep 15
}

function axiom_tester() {
    print_blue "===> 2. Clone snake"
    echo "$BRANCH_NAME"
    cd ../
    git clone -b "$BRANCH_NAME" https://github.com/axiomesh/snake.git
    cd snake
    print_blue "Start test"
    print_blue "===> 3. Start smoke test"
    npm install && npm run smoke-test
}

function axiom_test() {
    start_axiom_rbft
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

axiom_test