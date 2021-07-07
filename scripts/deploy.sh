set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
APP_VERSION=$(git describe --tag)
N=4

# help prompt message
function printHelp() {
  print_blue "Usage:  "
  echo "  deploy.sh [-a <bitxhub_addr>] [-r <if_recompile>] [-u <username>] [-p <build_path>]"
  echo "  - 'a' - the ip address of bitxhub node"
  echo "  - 'r' - if need to recompile locally"
  echo "  - 'u' - the username of remote linux server"
  echo "  - 'p' - the deploy path relative to HOME directory in linux server"
  echo "  deploy.sh -h (print this message)"
}

function splitWindow() {
  tmux splitw -v -p 50
  tmux splitw -h -p 50
  tmux selectp -t 0
  tmux splitw -h -p 50
}

function deploy() {
  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"
  print_blue "1. Generate config"
  for ((i = 1; i < $N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    mkdir -p "${root}"

    cp -rf "${CURRENT_PATH}"/certs/node${i}/certs "${root}"
    cp -rf "${PROJECT_PATH}"/config/* "${root}"

    echo " #!/usr/bin/env bash" >"${root}"/start.sh
    echo "./bitxhub --repo \$(pwd)" start >>"${root}"/start.sh
    x_replace 1s/1/${i}/ ${root}/network.toml
  done

  print_blue "2. Compile bitxhub"
  if [[ $IF_RECOMPILE == true ]]; then
    bash cross_compile.sh linux-amd64 ${PROJECT_PATH}
  else
    echo "Do not need compile"
  fi

  ips=$(echo $SERVER_ADDRESSES | tr "," "\n")
  ## prepare deploy package
  cd "${CURRENT_PATH}"
  for ((i = 1; i < $N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    cp ../bin/bitxhub_linux-amd64 ${root}/bitxhub

    j=1
    for ip in $ips; do
      x_replace s#ip4/127.0.0.1/tcp/400${j}#ip4/"$ip"/tcp/4001#g ${root}/network.toml
      ((j = j + 1))
    done
  done

  print_blue "3. Deploy bitxhub"

  i=1
  cd "${CURRENT_PATH}"
  for ip in $ips; do
    scp -r build/node${i} ${USERNAME}@$ip:~/
    ((i = i + 1))
  done

  tmux new -d -s deploy || (tmux kill-session -t deploy && tmux new -d -s deploy)

  for ((i = 0; i < $N / 4; i = i + 1)); do
    splitWindow
    tmux new-window
  done
  splitWindow

  i=0
  for ip in $ips; do
    tmux selectw -t $(($i / 4))
    tmux selectp -t $(($i % 4))
    ((i = i + 1))
    tmux send-keys "ssh -t $USERNAME@$ip 'export LD_LIBRARY_PATH=\${LD_LIBRARY_PATH}:\${HOME} && cd ~/node${i} && bash start.sh'" C-m
  done
  tmux selectw -t 0
  tmux attach-session -t deploy
}

while getopts "h?a:r:u:p:" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  a)
    SERVER_ADDRESSES=$OPTARG
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