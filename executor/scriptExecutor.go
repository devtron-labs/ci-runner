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

package executor

import (
	"fmt"
	cictx "github.com/devtron-labs/ci-runner/executor/context"
	util2 "github.com/devtron-labs/ci-runner/executor/util"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type ScriptExecutorImpl struct {
	cmdExecutor helper.CommandExecutor
}

type ScriptExecutor interface {
	RunScriptsV1(ciContext cictx.CiContext, outputPath string, bashScript string, script string, envVars map[string]string) error
	RunScripts(ciContext cictx.CiContext, string, scriptFileName string, script string, envInputVars map[string]string, outputVars []string) (map[string]string, error)
}

func NewScriptExecutorImpl(cmdExecutor helper.CommandExecutor) *ScriptExecutorImpl {
	return &ScriptExecutorImpl{
		cmdExecutor: cmdExecutor,
	}
}

func (impl *ScriptExecutorImpl) RunScriptsV1(ciContext cictx.CiContext, outputPath string, bashScript string, script string, envVars map[string]string) error {
	log.Println("running script commands")
	scriptTemplate := `#!/bin/sh
{{ range $key, $value := .envVr }}
export {{ $key }}='{{ $value }}' ;
{{ end }}
{{.script}}
`

	templateData := make(map[string]interface{})
	templateData["envVr"] = envVars
	templateData["script"] = script
	finalScript, err := util2.Tprintf(scriptTemplate, templateData)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}
	err = os.MkdirAll(outputPath, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}
	scriptPath := filepath.Join(outputPath, bashScript)
	file, err := os.Create(scriptPath)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()
	_, err = file.WriteString(finalScript)
	//log.Println(devtron, "final script ", finalScript) removed it shows some part on ui
	log.Println(util.DEVTRON, scriptPath)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	err = impl.cmdExecutor.RunCommand(ciContext, runScriptCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (impl *ScriptExecutorImpl) RunScripts(ciContext cictx.CiContext, workDirectory string, scriptFileName string, script string, envInputVars map[string]string, outputVars []string) (map[string]string, error) {
	log.Println("running script commands")
	envOutFileName := filepath.Join(workDirectory, fmt.Sprintf("%s_out.env", scriptFileName))

	//------------
	finalScript, err := prepareFinaleScript(script, outputVars, envOutFileName)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}
	//--------------
	scriptPath := filepath.Join(workDirectory, scriptFileName)
	file, err := os.Create(scriptPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer file.Close()
	_, err = file.WriteString(finalScript)
	//log.Println(util.DEVTRON, "final script ", finalScript) removed it shows some part on ui
	log.Println(util.DEVTRON, scriptPath)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}
	var inputEnvironmentVariable []string
	for k, v := range envInputVars {
		inputEnvironmentVariable = append(inputEnvironmentVariable, fmt.Sprintf("%s=%s", k, v))
	}
	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	runScriptCMD.Env = inputEnvironmentVariable
	err = impl.cmdExecutor.RunCommand(ciContext, runScriptCMD)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	envMap, err := godotenv.Read(envOutFileName)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return envMap, nil
}

// prepare final shell script to be executed
func prepareFinaleScript(script string, outputVars []string, envOutFileName string) (string, error) {
	scriptTemplate := `{{.script}}
> {{.envOutFileName}}
{{$envOutFileName := .envOutFileName}}
{{range .outputVars}} 
  printf "\n{{.}}=%s" "${{.}}" >> {{$envOutFileName}}
{{end}}
`
	templateData := make(map[string]interface{})
	templateData["script"] = script
	templateData["outputVars"] = outputVars
	templateData["envOutFileName"] = envOutFileName
	finalScript, err := util2.Tprintf(scriptTemplate, templateData)
	if err != nil {
		return "", err
	}
	return finalScript, nil
}

type executionConf struct {
	Script            string
	EnvInputVars      map[string]string
	ExposedPorts      map[int]int //map of host:container
	OutputVars        []string
	DockerImage       string
	command           string
	args              []string
	CustomScriptMount *helper.MountPath
	SourceCodeMount   *helper.MountPath
	ExtraVolumeMounts []*helper.MountPath
	OutputDirMount    []*helper.MountPath
	// system generate values
	scriptFileName      string //internal
	workDirectory       string
	EnvInputFileName    string // system generated
	EnvOutFileName      string // system generated
	EntryScriptFileName string // system generated
	RunCommandFileName  string // system generated
}

func RunScriptsInDocker(ciContext cictx.CiContext, impl *StageExecutorImpl, executionConf *executionConf) (map[string]string, error) {
	envInputFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_in.env", executionConf.scriptFileName))
	entryScriptFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_entry.sh", executionConf.scriptFileName))
	envOutFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_out.env", executionConf.scriptFileName))
	executionConf.RunCommandFileName = filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_run.sh", executionConf.scriptFileName))
	if executionConf.CustomScriptMount != nil && len(executionConf.Script) > 0 {
		customScriptMountFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_user_custom_script.sh", executionConf.scriptFileName))
		err := os.WriteFile(customScriptMountFileName, []byte(executionConf.Script), 0644) //TODO check mode with entry script
		if err != nil {
			log.Println(util.DEVTRON, err)
			return nil, err
		}
		executionConf.CustomScriptMount.SrcPath = customScriptMountFileName
	}

	executionConf.EnvInputFileName = envInputFileName
	executionConf.EntryScriptFileName = entryScriptFileName
	executionConf.EnvOutFileName = envOutFileName

	log.Println(util.DEVTRON, "envInputFilePath", envInputFileName)
	log.Println(util.DEVTRON, "EnvInputVars", executionConf.EnvInputVars)
	//Write env input vars to env file
	err := writeToEnvFile(executionConf.EnvInputVars, envInputFileName)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}

	entryScript, err := buildDockerEntryScript(executionConf.command, executionConf.args, executionConf.OutputVars)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}
	err = os.WriteFile(executionConf.EntryScriptFileName, []byte(entryScript), 0644) //TODO check mode with entry script
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}

	err = os.WriteFile(executionConf.EnvOutFileName, []byte(""), 0644) //TODO check mode with entry script
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}
	dockerRunCommand, err := buildDockerRunCommand(executionConf)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}

	fmt.Println(dockerRunCommand)
	//dockerRunCommand = "echo hello------;sleep 10; echo done------"
	err = os.WriteFile(executionConf.RunCommandFileName, []byte(dockerRunCommand), 0644)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}
	// docker run -it -v   -environment file  -p
	runScriptCMD := exec.Command("/bin/sh", executionConf.RunCommandFileName)
	//runScriptCMD.Env = inputEnvironmentVariable
	err = impl.cmdExecutor.RunCommand(ciContext, runScriptCMD)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	envMap, err := godotenv.Read(executionConf.EnvOutFileName)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return envMap, nil
}

func buildDockerEntryScript(command string, args []string, outputVars []string) (string, error) {
	entryTemplate := `#!/bin/sh
set -e
{{.command}} {{.args}}
> {{.envOutFileName}}
{{$envOutFileName := .envOutFileName}}
{{- range .outputVars -}} 
  printf "\n{{.}}=%s" "${{.}}" >> {{$envOutFileName}}
{{end -}}`

	templateData := make(map[string]interface{})
	templateData["args"] = strings.Join(args, " ")
	templateData["command"] = command
	templateData["envOutFileName"] = "/devtron_script/_out.env"
	templateData["outputVars"] = outputVars
	finalScript, err := util2.Tprintf(entryTemplate, templateData)
	if err != nil {
		return "", err
	}
	return finalScript, nil
}

func buildDockerRunCommand(executionConf *executionConf) (string, error) {
	cmdTemplate := `docker run --network host \
--env-file {{.EnvInputFileName}} \
-v {{.EntryScriptFileName}}:/devtron_script/_entry.sh \
-v {{.EnvOutFileName}}:/devtron_script/_out.env \
{{- if .SourceCodeMount }}
-v {{.SourceCodeMount.SrcPath}}:{{.SourceCodeMount.DstPath}} \
{{- end}}
{{- range .ExtraVolumeMounts }}
-v {{.SrcPath}}:{{.DstPath}} \
{{- end}}
{{- range .OutputDirMount }}
-v {{.SrcPath}}:{{.DstPath}} \
{{- end}}
{{- if .CustomScriptMount }}
-v {{ .CustomScriptMount.SrcPath}}:{{.CustomScriptMount.DstPath}} \
{{- end}}
{{- range $hostPort, $ContainerPort := .ExposedPorts }}
-p {{$hostPort}}:{{$ContainerPort}} \
{{- end }}
{{- .DockerImage}} \
/bin/sh /devtron_script/_entry.sh
`
	finalScript, err := util2.Tprintf(cmdTemplate, executionConf)
	if err != nil {
		return "", err
	}
	return finalScript, nil

}

// Writes input vars to env file
func writeToEnvFile(envMap map[string]string, filename string) error {
	content := formatEnvironmentVariables(envMap)
	file, err := os.Create(filename)
	if err != nil {
		log.Println(util.DEVTRON, "error while creating env file ", err)
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content + "\n")
	if err != nil {
		log.Println(util.DEVTRON, "error while writing env values to file ", err)
		return err
	}
	file.Sync()
	return err
}

// Filters values of env variables on the basis of type and inserts to slice of strings
func formatEnvironmentVariables(envMap map[string]string) string {
	lines := make([]string, 0, len(envMap))
	for k, v := range envMap {
		d, err := strconv.Atoi(v)
		if err != nil {
			//received string
			lines = append(lines, fmt.Sprintf(`%s=%s`, k, v))
		} else {
			//received integer
			lines = append(lines, fmt.Sprintf(`%s=%d`, k, d))
		}
	}
	return strings.Join(lines, util.NewLineChar)
}
