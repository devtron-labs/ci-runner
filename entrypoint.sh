#!/bin/sh

set -o pipefail

# Check the value of IN_APP_LOGGING environment variable
if [ "$IN_APP_LOGGING" = "true" ]; then
  # Run cirunner command with logging
  exec ./cirunner 2>&1 | tee main.log
else
  # Run cirunner command without logging
  exec ./cirunner
fi
#
#cirunner_pid=$!
#
## Register a function to forward SIGTERM to the cirunner process
#forward_sigterm() {
#  echo "Forwarding SIGTERM to cirunner process..."
#  kill -SIGTERM $cirunner_pid
#}
#
## Register the function to handle SIGTERM signal
#trap forward_sigterm SIGTERM