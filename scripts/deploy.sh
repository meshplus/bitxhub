set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
APP_VERSION=$(git describe --tag)

# help prompt message
function printHelp() {
  print_blue "Usage:  "
  echo "  deploy.sh [-a <bitxhub_addr>] [-n <node_num>] [-r <if_recompile>] [-u <username>] [-p <build_path>]"
  echo "  - 'a' - the ip address of bitxhub node"
  echo "  - 'n' - node number to be deployed in one server"
  echo "  - 'r' - if need to recompile locally"
  echo "  - 'u' - the username of remote linux server"
  echo "  - 'p' - the deploy path relative to HOME directory in linux server"
  echo "  deploy.sh -h (print this message)"
}

function deploy() {
  print_blue "1. Generate config"
  bash config.sh "$NODE_NUM"

  print_blue "2. Compile bitxhub"
  if [[ $IF_RECOMPILE == true ]]; then
    bash cross_compile.sh linux-amd64 ${PROJECT_PATH}
  else
    echo "Do not need compile"
  fi

  ## prepare deploy package
  cd "${CURRENT_PATH}"
  cp ../bin/bitxhub_linux-amd64 "${BUILD_PATH}"/bitxhub
  cp ../internal/plugins/build/*.so "${BUILD_PATH}"/
  tar cf build${APP_VERSION}.tar.gz build

  print_blue "3. Deploy bitxhub"
  cd "${CURRENT_PATH}"
  scp build${APP_VERSION}.tar.gz ${USERNAME}@"${BITXHUB_ADDR}":${SERVER_BUILD_PATH}

  ssh -t ${USERNAME}@"${BITXHUB_ADDR}" '
    cd '${SERVER_BUILD_PATH}'
    CURRENT_PATH=$(pwd)
    BUILD_PATH=${CURRENT_PATH}/build
    N='$NODE_NUM'

    function splitWindow() {
      tmux splitw -v -p 50
      tmux splitw -h -p 50
      tmux selectp -t 0
      tmux splitw -h -p 50
    }

    function start() {
      cd ${CURRENT_PATH}
      rm -rf build
      tar -xf build.tar.gz
      pkill -9 bitxhub
      tmux kill-session -t bitxhub
      tmux new -d -s bitxhub

      cd build
      for ((i=0;i<N/4;i=i+1)); do
        splitWindow
        tmux new-window
      done
      splitWindow
      for ((i = 0;i < N;i = i + 1)); do
        tmux selectw -t $(($i / 4))
        tmux selectp -t $(($i % 4))
        cp bitxhub ./node$(($i + 1))/
        if [ ! -d ./node$(($i + 1))/plugins ]; then
          mkdir ./node$(($i + 1))/plugins
        fi
        cp *.so ./node$(($i + 1))/plugins/
        tmux send-keys "cd ${BUILD_PATH} && ./node$(($i + 1))/bitxhub --repo=${BUILD_PATH}/node$(($i + 1)) start" C-m
      done
      tmux selectw -t 0
    }

    start

    tmux attach-session -t bitxhub
'
}

while getopts "h?a:n:r:u:p:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  a)
    BITXHUB_ADDR=$OPTARG
    ;;
  n)
    NODE_NUM=$OPTARG
    ;;
  r)
    IF_RECOMPILE=$OPTARG
    ;;
  u)
    USERNAME=$OPTARG
    ;;
  p)
    SERVER_BUILD_PATH=/home/${USERNAME}/$OPTARG
    ;;
  esac
done

deploy
