#!/usr/bin/env bash

set -e

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
CERT_PATH=${CURRENT_PATH}/cert
N=$1

mkdir -p "${CERT_PATH}"
cd "${CERT_PATH}"

## Generate ca private key and cert
premo cert ca

for ((i = 1; i < N + 1; i = i + 1)); do
  mkdir -p "${CERT_PATH}"/node${i}
  cd "${CERT_PATH}"/node${i}
  premo cert issue --name agency --priv ../ca.priv --cert ../ca.cert --org=Hyperchain
  premo cert issue --name node --priv ./agency.priv --cert ./agency.cert --org=Agency${i}
done
