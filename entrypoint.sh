#!/bin/sh

set -o pipefail

# Check the value of IN_APP_LOGGING environment variable
if [ "$IN_APP_LOGGING" = "true" ]; then
  # Run cirunner command with logging
#  exec ./cirunner 2>&1 | tee main.log
#  ./cirunner 2>&1 | { tee main.log & echo $! > cirunner_pid.txt; }
  { ./cirunner 2>&1 & echo $! > cirunner_pid.txt; } | tee main.log
else
  # Run cirunner command without logging
  exec ./cirunner
fi

# Read the cirunner PID from cirunner_pid.txt
cirunner_pid=$(cat cirunner_pid.txt)
rm cirunner_pid.txt

# Register a function to forward SIGTERM to the cirunner process
forward_sigterm() {
  echo "Forwarding SIGTERM to cirunner process..."
  kill -SIGTERM $cirunner_pid
}

# Register the function to handle SIGTERM signal
trap forward_sigterm SIGTERM

## Wait for the cirunner process to complete
#wait $cirunner_pid
#
## Perform any additional cleanup if needed
#echo "Script cleanup logic here..."



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