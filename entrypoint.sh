#!/bin/sh

set -o pipefail

cleanup() {
  echo "Cleaning up.."
  echo "$PID of cirunner: "
  echo "$cirunner_pid"
  # Send SIGTERM to the cirunner process
  kill -TERM "$cirunner_pid"
}

# Check the value of IN_APP_LOGGING environment variable
if [ "$IN_APP_LOGGING" = "true" ]; then
  trap 'cleanup' SIGTERM

  # Start cirunner in background & capture its pid

  ./cirunner 2>&1 | {
    tee main.log &
    tee_pid=$!

    #Capture pid of cirunner
    cirunner_pid=$!

    #Wait for both cirunner and tee processes to finish
    wait "$cirunner_pid"
    wait "$tee_pid"
    }
else
  # Run cirunner command without logging
  exec ./cirunner
fi


