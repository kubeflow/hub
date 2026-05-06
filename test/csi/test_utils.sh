#!/bin/bash

set -e
set -o xtrace

# Function to check if the port is ready
wait_for_port() {
  local port=$1
  while ! nc -z localhost $port; do
    sleep 0.1
  done
}

repeat_cmd_until() {
  local cmd=$1
  local max_wait_secs=$2
  local debug_cmd=$3
  shift 3
  local condition_args=("$@")

  local interval_secs=2
  local start_time=$(date +%s)
  local output

  while true; do

    current_time=$(date +%s)
    if (( (current_time - start_time) > max_wait_secs )); then
      echo "Waited for expression '$cmd' to satisfy condition '${condition_args[*]}' for $max_wait_secs seconds without luck. Returning with error."
      if [ -n "$debug_cmd" ]; then
        echo "Running debug command: $debug_cmd"
        eval "$debug_cmd"
      fi
      return 1
    fi

    output=$(eval "$cmd")

    if [ "$output" "${condition_args[@]}" ]; then
      break
    else
      sleep $interval_secs
    fi
  done
}
