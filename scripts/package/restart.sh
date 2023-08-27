#!/usr/bin/env bash
set -e

base_dir=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)

bash stop.sh

if [ -n "$1" ]; then
    mv axiom axiom.bak
    # $1 is new axiom path
    cp $1 axiom
fi

nohup bash start.sh > /dev/null 2>&1 &