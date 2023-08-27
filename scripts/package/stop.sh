#!/usr/bin/env bash
set -e

base_dir=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)

wait_axiom_quit_time=50
wait_axiom_quit_interval=0.2

function wait_axiom_quit(){
  for ((i = 1; i <= ${wait_axiom_quit_time}; i = i + 1)); do
    process_cnt=$(ps -p${1} -o pid,comm | awk 'END{print NR}')
    if [ "${process_cnt}" == "2" ]; then
      sleep ${wait_axiom_quit_interval}
    else
      echo "axiom quit"
      return
    fi
  done
  echo "stop axiom failed, wait process quit timeout, use kill -9 forced stop"
  kill -9 ${1}
}

function stop(){
  if [ -f ${base_dir}/axiom.pid ]; then
    pid=$(cat ${base_dir}/axiom.pid)
    if [ "${pid}" ]; then
      cmd_name=$(ps -p${pid} -o pid,comm | awk 'NR==2{print $2}')
      if [[ "${cmd_name}" =~ "axiom" ]]; then
        kill ${pid}
        echo "stop axiom, pid: ${pid}"
        wait_axiom_quit ${pid}
        return
      fi
    fi
  fi
  echo "stop axiom, node is not running"
}

stop