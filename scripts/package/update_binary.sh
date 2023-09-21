#! /bin/bash
set -e

if [ ! -f $1 ]; then
  echo "error: new binary($1) does not exist"
  exit 1
fi

base_dir=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)

# backup old binary
old_binary=${base_dir}/tools/bin/axiom-ledger-$(date +%Y-%m-%d-%H-%M-%S).bak
cp -f ${base_dir}/tools/bin/axiom-ledger ${old_binary}
cp -f $1 ${base_dir}/tools/bin/axiom-ledger

echo "backup old binary to ${old_binary}"
echo "new binary:"
${base_dir}/axiom-ledger version