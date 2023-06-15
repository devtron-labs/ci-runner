#!/bin/sh

set -o pipefail

cleanup() {
  echo "Cleaning up.."
  echo 'PID of cirunner: '
  echo $cirunner_pid
  # Send SIGTERM to the cirunner process
  kill -TERM "$cirunner_pid"

}

# Check the value of IN_APP_LOGGING environment variable
if [ "$IN_APP_LOGGING" = "true" ]; then
  # Run cirunner command with logging
#  exec ./cirunner 2>&1 | tee main.log
   trap 'cleanup' SIGTERM
  { ./cirunner 2>&1 & echo $! > cirunner_pid.txt; } | tee main.log &
  # Read the cirunner PID from cirunner_pid.txt
  cirunner_pid=$(cat cirunner_pid.txt)
  echo 'PID of cirunner: '
  echo $cirunner_pid
  wait "$cirunner_pid"
#  rm cirunner_pid.txt
else
  # Run cirunner command without logging
  exec ./cirunner
fi


