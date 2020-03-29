#!/usr/bin/env bash

set -e

# integration test script
CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
N=$1

sh build.sh "$N"

cd "${PROJECT_PATH}"
make install

bitxhub version
for ((i = 1; i < N + 1; i = i + 1)); do
  echo "Start node${i}"
  nohup bitxhub --repo="${BUILD_PATH}"/node${i} start &
done
