set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build

USERNAME=hyperchain
SERVER_BUILD_PATH="/home/hyperchain/bitxhub"
N=$1
IP=$2
RECOMPILE=$3

if [ "$#" -ne 3 ]; then
  echo "Illegal number of parameters"
  exit
fi

while getopts "h?n:r:" opt; do
  case "$opt" in
  h | \?)
    help
    exit 0
    ;;
  n)
    N=$OPTARG
    ;;
  r)
    RECOMPILE=$OPTARG
    ;;
  esac
done

function build() {
  bash config.sh "$N"
}

function compile() {
  if [[ $RECOMPILE == true ]]; then
    cd "${PROJECT_PATH}"
    make build-linux
  else
    echo "Do not need compile"
  fi
}

function prepare() {
  cd "${CURRENT_PATH}"
  cp ../bin/bitxhub_linux-amd64 "${BUILD_PATH}"/bitxhub
  cp ../internal/plugins/build/*.so "${BUILD_PATH}"/
  tar cf build.tar.gz build
}

function deploy() {
  cd "${CURRENT_PATH}"
  #ssh-copy-id -i ~/.ssh/id_rsa.pub hyperchain@${IP}
  scp build.tar.gz ${USERNAME}@"${IP}":${SERVER_BUILD_PATH}
  scp boot_bitxhub.sh ${USERNAME}@"${IP}":${SERVER_BUILD_PATH}

  ssh -t ${USERNAME}@"${IP}" '
    cd '${SERVER_BUILD_PATH}'
    bash boot_bitxhub.sh 4
    tmux attach-session -t bitxhub
'
}

# help prompt message
function help() {
  echo "deploy.sh helper:"
  echo "  -h, --help:     show the help for this bash script"
  echo "  -n, --number:   bitxhub node to be started"
  echo "  -r, --recompile: need compile or not"
  echo "---------------------------------------------------"
  echo "Example for deploy 4 node in server:"
  echo "./deploy.sh -n 4 -r 1"
}

print_blue "1. Generate config"
build

print_blue "2. Compile bitxhub"
compile
prepare

print_blue "3. Deploy bitxhub"
deploy
