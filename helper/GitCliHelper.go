package helper

import (
	"fmt"
	"github.com/devtron-labs/ci-runner/util"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitUtil struct {
}

func NewGitUtil() *GitUtil {
	return &GitUtil{}
}

const GIT_AKS_PASS = "/git-ask-pass.sh"

func (impl *GitUtil) Fetch(gitContext GitContext, rootDir string) (response, errMsg string, err error) {
	log.Println(util.DEVTRON, "git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	output, errMsg, err := impl.RunCommandWithCred(cmd, gitContext.auth.Username, gitContext.auth.Password)
	log.Println(util.DEVTRON, "fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, "", nil
}

func (impl *GitUtil) Checkout(gitContext GitContext, rootDir string, checkout string) (response, errMsg string, err error) {
	log.Println(util.DEVTRON, "git checkout ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "checkout", checkout, "--force")
	output, errMsg, err := impl.RunCommandWithCred(cmd, gitContext.auth.Username, gitContext.auth.Password)
	log.Println(util.DEVTRON, "checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, "", nil
}

func (impl *GitUtil) RunCommandWithCred(cmd *exec.Cmd, userName, password string) (response, errMsg string, err error) {
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_AKS_PASS),
		fmt.Sprintf("GIT_USERNAME=%s", userName), // ignored
		fmt.Sprintf("GIT_PASSWORD=%s", password), // this value is used
	)
	return impl.RunCommand(cmd)
}

func (impl *GitUtil) RunCommand(cmd *exec.Cmd) (response, errMsg string, err error) {
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

func (impl *GitUtil) Clone(gitContext GitContext, prj CiProjectDetails) (response, errMsg string, err error) {
	rootDir := filepath.Join(util.WORKINGDIR, prj.CheckoutPath)
	remoteUrl := prj.GitRepository
	err = impl.Init(rootDir, remoteUrl, false)
	if err != nil {
		return "", "", err
	}

	response, errMsg, err = impl.Fetch(gitContext, rootDir)
	return response, errMsg, err
}

// setting user.name and user.email as for non-fast-forward merge, git ask for user.name and email
func (impl *GitUtil) Merge(rootDir string, commit string) (response, errMsg string, err error) {
	log.Println(util.DEVTRON, "git merge ", "location", rootDir)
	command := "cd " + rootDir + " && git config user.email git@devtron.com && git config user.name Devtron && git merge " + commit + " --no-commit"
	cmd := exec.Command("/bin/sh", "-c", command)
	output, errMsg, err := impl.RunCommand(cmd)
	log.Println(util.DEVTRON, "merge output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitUtil) RecursiveFetchSubmodules(rootDir string) (response, errMsg string, error error) {
	log.Println(util.DEVTRON, "git recursive fetch submodules ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "submodule", "update", "--init", "--recursive")
	output, eMsg, err := impl.runCommandForSuppliedNullifiedEnv(cmd, false)
	log.Println(util.DEVTRON, "recursive fetch submodules output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

func (impl *GitUtil) UpdateCredentialHelper(rootDir string) (response, errMsg string, error error) {
	log.Println(util.DEVTRON, "git credential helper store ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "config", "--global", "credential.helper", "store")
	output, eMsg, err := impl.runCommandForSuppliedNullifiedEnv(cmd, false)
	log.Println(util.DEVTRON, "git credential helper store output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

func (impl *GitUtil) UnsetCredentialHelper(rootDir string) (response, errMsg string, error error) {
	log.Println(util.DEVTRON, "git credential helper unset ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "config", "--global", "--unset", "credential.helper")
	output, eMsg, err := impl.runCommandForSuppliedNullifiedEnv(cmd, false)
	log.Println(util.DEVTRON, "git credential helper unset output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, eMsg, err
}

func (impl *GitUtil) GitCheckout(gitContext GitContext, gitCli *GitUtil, checkoutPath string, targetCheckout string, authMode AuthMode, fetchSubmodules bool, gitRepository string) (errMsg string, error error) {

	rootDir := filepath.Join(util.WORKINGDIR, checkoutPath)

	// checkout target hash
	_, eMsg, cErr := gitCli.Checkout(gitContext, rootDir, targetCheckout)
	if cErr != nil {
		return eMsg, cErr
	}

	log.Println(util.DEVTRON, " fetchSubmodules ", fetchSubmodules, " authMode ", authMode)

	if fetchSubmodules {
		httpsAuth := (authMode == AUTH_MODE_USERNAME_PASSWORD) || (authMode == AUTH_MODE_ACCESS_TOKEN)
		if httpsAuth {
			// first remove protocol
			modifiedUrl := strings.ReplaceAll(gitRepository, "https://", "")
			// for bitbucket - if git repo url is started with username, then we need to remove username
			if strings.Contains(modifiedUrl, "bitbucket.org") && !strings.HasPrefix(modifiedUrl, "bitbucket.org") {
				modifiedUrl = modifiedUrl[strings.Index(modifiedUrl, "bitbucket.org"):]
			}
			// build url
			modifiedUrl = "https://" + gitContext.auth.Username + ":" + gitContext.auth.Password + "@" + modifiedUrl

			_, errMsg, cErr = gitCli.UpdateCredentialHelper(rootDir)
			if cErr != nil {
				return errMsg, cErr
			}

			cErr = util.CreateGitCredentialFileAndWriteData(modifiedUrl)
			if cErr != nil {
				return "Error in creating git credential file", cErr
			}

		}

		_, errMsg, cErr = gitCli.RecursiveFetchSubmodules(rootDir)
		if cErr != nil {
			return errMsg, cErr
		}

		// cleanup

		if httpsAuth {
			_, errMsg, cErr = gitCli.UnsetCredentialHelper(rootDir)
			if cErr != nil {
				return errMsg, cErr
			}

			// delete file (~/.git-credentials) (which was created above)
			cErr = util.CleanupAfterFetchingHttpsSubmodules()
			if cErr != nil {
				return "", cErr
			}
		}
	}

	return "", nil

}
