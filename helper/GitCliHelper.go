package helper

import (
	"fmt"
	"github.com/devtron-labs/ci-runner/util"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"log"
	"os"
	"os/exec"
	"strings"
)

type GitUtil struct {
}

func NewGitUtil() *GitUtil {
	return &GitUtil{}
}

const GIT_AKS_PASS = "/git-ask-pass.sh"

// Fetch uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) Fetch(gitContext GitContext, rootDir string) (response, errMsg string, err error) {
	log.Println(util.DEVTRON, "git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.auth.Username, gitContext.auth.Password)
	log.Println(util.DEVTRON, "fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, "", nil
}

// Checkout uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) Checkout(rootDir string, checkout string) (response, errMsg string, err error) {
	log.Println(util.DEVTRON, "git checkout ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "checkout", checkout, "--force")
	output, errMsg, err := impl.runCommand(cmd)
	log.Println(util.DEVTRON, "checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, "", nil
}

// runCommandWithCred uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3))  |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) runCommandWithCred(cmd *exec.Cmd, userName, password string) (response, errMsg string, err error) {
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_AKS_PASS),
		fmt.Sprintf("GIT_USERNAME=%q", userName), // ignored; %q is used intentionally to sanitise the username
		fmt.Sprintf("GIT_PASSWORD=%q", password), // this value is used; %q is used intentionally to sanitise the password
	)
	return impl.runCommand(cmd)
}

func (impl *GitUtil) runCommand(cmd *exec.Cmd) (response, errMsg string, err error) {
	return impl.runCommandForSuppliedNullifiedEnv(cmd, true)
}

func (impl *GitUtil) runCommandForSuppliedNullifiedEnv(cmd *exec.Cmd, setHomeEnvToNull bool) (response, errMsg string, err error) {
	if setHomeEnvToNull {
		cmd.Env = append(cmd.Env, "HOME=/dev/null")
	}
	// https://stackoverflow.com/questions/18159704/how-to-debug-exit-status-1-error-when-running-exec-command-in-golang
	// in CombinedOutput, both stdOut and stdError are returned in single output
	outBytes, err := cmd.CombinedOutput()
	output := string(outBytes)
	output = strings.Replace(output, "\n", "", -1)
	output = strings.TrimSpace(output)
	if err != nil {
		exErr, ok := err.(*exec.ExitError)
		if !ok {
			return "", output, err
		}
		errOutput := string(exErr.Stderr)
		return "", fmt.Sprintf("%s\n%s", output, errOutput), err
	}
	return output, "", nil
}

func (impl *GitUtil) Init(rootDir string, remoteUrl string, isBare bool) error {

	//-----------------

	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return err
	}
	repo, err := git.PlainInit(rootDir, isBare)
	if err != nil {
		return err
	}
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{remoteUrl},
	})
	return err
}

func (impl *GitUtil) Clone(gitContext GitContext, rootDir string, remoteUrl string) (response, errMsg string, err error) {
	err = impl.Init(rootDir, remoteUrl, false)
	if err != nil {
		return "", "", err
	}

	response, errMsg, err = impl.Fetch(gitContext, rootDir)
	return response, errMsg, err
}

// Merge sets user.name and user.email as for non-fast-forward merge, git ask for user.name and email |
// Merge uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) Merge(rootDir string, commit string) (response, errMsg string, err error) {
	log.Println(util.DEVTRON, "git merge ", "location", rootDir)
	command := util.NewCommand("cd", rootDir, "&&", "git", "config", "user.email", "git@devtron.com", "&&", "git", "config", "user.name", "Devtron", "&&", "git", "merge", commit, "--no-commit")
	cmd := exec.Command("/bin/sh", command.GetCommandToBeExecuted("-c")...)
	output, errMsg, err := impl.runCommand(cmd)
	log.Println(util.DEVTRON, "merge output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

// RecursiveFetchSubmodules uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) RecursiveFetchSubmodules(rootDir string) (response, errMsg string, error error) {
	log.Println(util.DEVTRON, "git recursive fetch submodules ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "submodule", "update", "--init", "--recursive")
	output, eMsg, err := impl.runCommandForSuppliedNullifiedEnv(cmd, false)
	log.Println(util.DEVTRON, "recursive fetch submodules output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

// UpdateCredentialHelper uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) UpdateCredentialHelper(rootDir string) (response, errMsg string, error error) {
	log.Println(util.DEVTRON, "git credential helper store ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "config", "--global", "credential.helper", "store")
	output, eMsg, err := impl.runCommandForSuppliedNullifiedEnv(cmd, false)
	log.Println(util.DEVTRON, "git credential helper store output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

// UnsetCredentialHelper uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func (impl *GitUtil) UnsetCredentialHelper(rootDir string) (response, errMsg string, error error) {
	log.Println(util.DEVTRON, "git credential helper unset ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "config", "--global", "--unset", "credential.helper")
	output, eMsg, err := impl.runCommandForSuppliedNullifiedEnv(cmd, false)
	log.Println(util.DEVTRON, "git credential helper unset output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}
