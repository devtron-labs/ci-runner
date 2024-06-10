/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// SpawnProcessWithLogging
// This method handles the logic for maintaining a local log file for log archival and handles SIGTERM propagation.
// The subprocess spawned will bypass this function and execute the main logic
func SpawnProcessWithLogging() {

	// Create an in-memory pipe
	pr, pw := io.Pipe()

	// Create the cirunner command
	cirunnerCmd := exec.Command(CiRunnerCommand)

	// combining stdout and stderr
	cirunnerCmd.Stdout = pw
	cirunnerCmd.Stderr = pw

	// Create the tee command
	teeCmd := exec.Command(TeeCommand, LogFileName)
	teeCmd.Stdin = pr
	teeCmd.Stdout = os.Stdout

	// Start cirunner
	cirunnerCmd.Start()
	// Start tee
	teeCmd.Start()

	// Create a channel to receive the SIGTERM signal
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	go func() {
		log.Println(DEVTRON, "SIGTERM listener started in parent process!")
		receivedSignal := <-sigTerm
		log.Println(DEVTRON, "signal received in parent process: ", receivedSignal)

		// sending SIGTERM to the subprocess
		cirunnerCmd.Process.Signal(syscall.SIGTERM)
	}()

	// wait until cirunner subprocess completes execution
	processState, _ := cirunnerCmd.Process.Wait()
	exitCode := processState.ExitCode()

	// Close write end of the pipe
	pw.Close()

	// exit with exit code returned by subprocess
	os.Exit(exitCode)
}
