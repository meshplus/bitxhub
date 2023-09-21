#! /bin/bash
set -e

shell_dir=$(cd $(dirname ${BASH_SOURCE[0]}); pwd)
base_dir=`dirname ${shell_dir}`
app_name=axiom-ledger
bin_path=${shell_dir}/bin/${app_name}
pid_file=${base_dir}/running.pid
wait_process_exit_check_time=50
wait_process_exit_check_interval=0.2

function wait_process_exit(){
  for ((i = 1; i <= ${wait_process_exit_check_time}; i = i + 1)); do
    process_cnt=$(ps -p${1} -o pid,comm | awk 'END{print NR}')
    if [ "${process_cnt}" == "2" ]; then
      sleep ${wait_process_exit_check_interval}
    else
      echo "${app_name} quit"
      return
    fi
  done
  echo "stop ${app_name} failed, wait process quit timeout, use kill -9 forced stop"
  kill -9 ${1}
}

# get_running_pid
function get_running_pid(){
  if [ -f ${pid_file} ]; then
    pid=$(cat ${pid_file})
    if [ "${pid}" ]; then
      cmd_name=$(ps w -p${pid} -o command | awk 'END{print $1}')
      if [[ "${cmd_name}" == *${app_name} ]]; then
        echo ${pid}
      fi
    fi
  fi
}

function start(){
  pid=`get_running_pid`
  if [[ ! "${pid}" == "" ]]; then
    echo "${app_name} is running, pid: ${pid}"
    return
  fi

  ${bin_path} config check
  nohup ${bin_path} start >/dev/null 2>&1 &
  echo "start ${app_name}, pid: $!"
}

function stop(){
  pid=`get_running_pid`
  if [[ ! "${pid}" == "" ]]; then
    kill ${pid}
    echo "stop ${app_name}, pid: ${pid}"
    wait_process_exit ${pid}
    rm -f ${pid_file}
  else
    echo "stop ${app_name}, ${app_name} is not running"
  fi
}

function restart(){
  stop
  start
}

function status(){
  pid=`get_running_pid`
  if [[ ! "${pid}" == "" ]]; then
    echo "${app_name} is running, pid: ${pid}"
  else
    echo "${app_name} is stopped"
  fi
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    restart
    ;;
  status)
    status
    ;;
  *)
    echo "Usage: ./control.sh {start|stop|restart|status}"
    exit 1
esac

exit $?