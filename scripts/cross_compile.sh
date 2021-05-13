#!/usr/bin/env bash
PROJECT_PATH=$(dirname "$(pwd)")
BIN_PATH=${PROJECT_PATH}/bin
set -e

source x.sh

# $1 is arch, $2 is source code path
case $1 in
linux-amd64)
  print_blue "Compile for linux/amd64"
  if [ -z "$(docker image inspect golang:1.13)" ]; then
    docker pull golang:1.13
  else
    print_blue "golang:1.13 image already exist"
  fi

  if [ "$(docker container ls -a | grep -c bitxhub_linux)" -ge 1 ];then
    print_blue "golang:1.13 container already exist"
    rm -f "${BIN_PATH}"/bitxhub_linux-amd64
    docker restart bitxhub_linux
    docker logs bitxhub_linux -f --tail "0"
  else
    docker run --name bitxhub_linux -t \
      -v $2:/code/bitxhub \
      -v ~/.ssh:/root/.ssh \
      -v ~/.gitconfig:/root/.gitconfig \
      -v $GOPATH/pkg/mod:$GOPATH/pkg/mod \
      golang:1.13 \
      /bin/bash /code/bitxhub/scripts/compile.sh
  fi
  ;;
*)
  print_red "Other architectures are not supported yet"
  ;;
esac
