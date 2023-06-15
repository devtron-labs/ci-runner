#!/bin/sh

set -o pipefail

cleanup() {
  echo "Cleaning up.."
  echo 'PID of cirunner: '
  echo $cirunner_pid
  echo 'PID of tee: '
  echo $tee_pid

  # Send SIGTERM to the cirunner process
  kill -TERM "$cirunner_pid"

  # Send SIGTERM to the tee process
  kill -TERM "$tee_pid"
}

# Check the value of IN_APP_LOGGING environment variable
if [ "$IN_APP_LOGGING" = "true" ]; then
  # Run cirunner command with logging
#  exec ./cirunner 2>&1 | tee main.log
   trap 'cleanup' SIGTERM
  ./cirunner 2>&1 | tee main.log & tee_pid=$!
    #Capture pid of cirunner
    cirunner_pid=$!
    echo 'PID of cirunner: '
    echo $cirunner_pid
    echo 'PID of tee: '
    echo $tee_pid
    wait "$cirunner_pid"
else
  # Run cirunner command without logging
  exec ./cirunner
fi


