#! /bin/bash
set -e

base_dir=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)
env_file=${base_dir}/.env.sh
if [ -f ${env_file} ]; then
  source ${env_file}
fi
export AXIOM_LEDGER_PATH=${base_dir}

${base_dir}/tools/control.sh restart