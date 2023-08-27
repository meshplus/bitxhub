#!/usr/bin/env bash
set -e

base_dir=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)

./axiom --repo ${base_dir} config generate