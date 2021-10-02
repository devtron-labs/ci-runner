package main

import (
	"fmt"
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
	return &GitUtil{
	}
}

const GIT_AKS_PASS = "/git-ask-pass.sh"

func (impl *GitUtil) Fetch(rootDir string, username string, password string) (response, errMsg string, err error) {
	log.Println(devtron, "git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	output, errMsg, err := impl.runCommandWithCred(cmd, username, password)
	log.Println(devtron, "fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, "", nil
}

func (impl *GitUtil) Checkout(rootDir string, checkoutCommit string) (response, errMsg string, err error) {
	log.Println(devtron, "git checkout ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "checkout", checkoutCommit, "--force")
	output, errMsg, err := impl.runCommand(cmd)
	log.Println(devtron, "checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, "", nil
}

func (impl *GitUtil) runCommandWithCred(cmd *exec.Cmd, userName, password string) (response, errMsg string, err error) {
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_AKS_PASS),
		fmt.Sprintf("GIT_USERNAME=%s", userName), // ignored
		fmt.Sprintf("GIT_PASSWORD=%s", password), // this value is used
	)
	return impl.runCommand(cmd)
}

func (impl *GitUtil) runCommand(cmd *exec.Cmd) (response, errMsg string, err error) {
	cmd.Env = append(cmd.Env, "HOME=/dev/null")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		exErr, ok := err.(*exec.ExitError)
		if !ok {
			return "", "", err
		}
		errOutput := string(exErr.Stderr)
		return "", errOutput, err
	}
	output := string(outBytes)
	output = strings.Replace(output, "\n", "", -1)
	output = strings.TrimSpace(output)
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

func (impl *GitUtil) Clone(rootDir string, remoteUrl string, username string, password string) (response, errMsg string, err error) {
	err = impl.Init(rootDir, remoteUrl, false)
	if err != nil {
		return "", "", err
	}

	response, errMsg, err = impl.Fetch(rootDir, username, password)
	return response, errMsg, err
}

// setting user.name and user.email as for non-fast-forward merge, git ask for user.name and email
func (impl *GitUtil) Merge(rootDir string, commit string) (response, errMsg string, err error) {
	log.Println(devtron, "git merge ", "location", rootDir)
	command := "cd " + rootDir + " && git config user.email git@devtron.com && git config user.name Devtron && git merge " + commit + " --no-commit"
	cmd := exec.Command("/bin/sh", "-c", command)
	output, errMsg, err := impl.runCommand(cmd)
	log.Println(devtron, "merge output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitUtil) RecursiveFetchSubmodules(rootDir string) (response, errMsg string, error error) {
	log.Println(devtron, "git recursive fetch submodules ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "submodule", "update", "--init", "--recursive")
	output, eMsg, err := impl.runCommand(cmd)
	log.Println(devtron, "recursive fetch submodules output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

func (impl *GitUtil) UpdateCredentialHelper(rootDir string) (response, errMsg string, error error) {
	log.Println(devtron, "git credential helper store ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "config", "--global", "credential.helper", "store")
	output, eMsg, err := impl.runCommand(cmd)
	log.Println(devtron, "git credential helper store output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

func (impl *GitUtil) UnsetCredentialHelper(rootDir string) (response, errMsg string, error error) {
	log.Println(devtron, "git credential helper unset ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "config", "--global", "--unset", "credential.helper")
	output, eMsg, err := impl.runCommand(cmd)
	log.Println(devtron, "git credential helper unset output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}