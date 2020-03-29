#!/usr/bin/env bash

set -e

source x.sh

CURRENT_PATH=$(pwd)
PROJECT_PATH=$(dirname "${CURRENT_PATH}")
BUILD_PATH=${CURRENT_PATH}/build
CONFIG_PATH=${PROJECT_PATH}/config
N=$1

function prepare() {
  rm -rf "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"
  mkdir "${BUILD_PATH}"/certs
}

function generate_certs() {
  for ((i = 1; i < N + 1; i = i + 1)); do
    if [[ $i -le 4 ]]; then
      cp -rf "${CURRENT_PATH}"/certs/node${i} "${BUILD_PATH}"/certs
    else
      certs_path=${BUILD_PATH}/certs/node${i}/certs
      mkdir -p "${certs_path}"
      cp "${CURRENT_PATH}"/certs/ca.cert "${CURRENT_PATH}"/certs/agency.cert "${certs_path}"
      premo cert priv --name node --target "${certs_path}"
      premo cert csr --key "${certs_path}"/node.priv --org Node${i} --target "${certs_path}"
      premo cert issue --csr "${certs_path}"/node.csr \
        --key "${CURRENT_PATH}"/certs/ca.priv \
        --cert "${CURRENT_PATH}"/certs/ca.cert \
        --target "${certs_path}"

      premo key convert --path "${certs_path}"/node.priv --target "${BUILD_PATH}"/certs/node${i}
    fi
  done
}

# Generate config
function generate() {
  for ((i = 1; i < N + 1; i = i + 1)); do
    root=${BUILD_PATH}/node${i}
    mkdir -p "${root}"
    cp -rf "${BUILD_PATH}"/certs/node${i}/* "${root}"
    cp -rf "${CONFIG_PATH}"/* "${root}"

    echo "#!/usr/bin/env bash" >"${root}"/start.sh
    echo "./bitxhub --root \$(pwd)" start >>"${root}"/start.sh

    bitxhubConfig=${root}/bitxhub.toml
    networkConfig=${root}/network.toml
    x_replace "s/60011/6001${i}/g" "${bitxhubConfig}"
    x_replace "s/60011/6001${i}/g" "${bitxhubConfig}"
    x_replace "s/9091/909${i}/g" "${bitxhubConfig}"
    x_replace "s/53121/5312${i}/g" "${bitxhubConfig}"
    x_replace "s/9091/909${i}/g" "${root}"/api
    x_replace "1s/1/${i}/" "${networkConfig}"
  done
}

print_green "Generating $1 nodes configuration..."
prepare
generate_certs
generate
