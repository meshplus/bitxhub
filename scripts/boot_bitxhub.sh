CURRENT_PATH=$(pwd)
BUILD_PATH=${CURRENT_PATH}/build
N=$1

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
    mkdir ./node$(($i + 1))/plugins
    cp *.so ./node$(($i + 1))/plugins/
    tmux send-keys "cd ${BUILD_PATH} && ./node$(($i + 1))/bitxhub --repo=${BUILD_PATH}/node$(($i + 1)) start" C-m
  done
  tmux selectw -t 0
}

start
