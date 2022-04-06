package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunScripts(workDirectory string, scriptFileName string, script string, envInputVars map[string]string, outputVars []string) (map[string]string, error) {
	log.Println("running script commands")
	envOutFileName := filepath.Join(workDirectory, fmt.Sprintf("%s_out.env", scriptFileName))

	//------------
	finalScript, err := prepareFinaleScript(script, outputVars, envOutFileName)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	//--------------
	err = os.MkdirAll(workDirectory, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	scriptPath := filepath.Join(workDirectory, scriptFileName)
	file, err := os.Create(scriptPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer file.Close()
	_, err = file.WriteString(finalScript)
	//log.Println(devtron, "final script ", finalScript) removed it shows some part on ui
	log.Println(devtron, scriptPath)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	var inputEnvironmentVariable []string
	for k, v := range envInputVars {
		inputEnvironmentVariable = append(inputEnvironmentVariable, fmt.Sprintf("%s=%s", k, v))
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	runScriptCMD.Env = inputEnvironmentVariable
	err = RunCommand(runScriptCMD)
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

//prepare final shell script to be executed
func prepareFinaleScript(script string, outputVars []string, envOutFileName string) (string, error) {
	scriptTemplate := `
#!/bin/sh
set -e
set -o pipefail
{{.script}}
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
	finalScript, err := Tprintf(scriptTemplate, templateData)
	if err != nil {
		return "", err
	}
	return finalScript, nil
}

type executionConf struct {
	Script                  string
	ScriptLocation          string
	ScriptMountLocation     string
	EnvInputVars            map[string]string
	ExposedPorts            map[int]int
	OutputVars              []string
	DockerImage             string
	MountCode               bool
	SourceCodeLocation      string
	SourceCodeMountLocation string
	command                 string
	args                    []string
	scriptFileName          string
	// system generate values
	workDirectory       string
	EnvInputFileName    string // system generated
	EnvOutFileName      string // system generated
	EntryScriptFileName string // system generated
	RunCommandFileName  string // system generated
}

func RunScriptsInDocker(executionConf *executionConf) (map[string]string, error) {
	envInputFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_in.env", executionConf.scriptFileName))
	entryScriptFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_entry.sh", executionConf.scriptFileName))
	envOutFileName := filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_out.env", executionConf.scriptFileName))
	executionConf.RunCommandFileName = filepath.Join(executionConf.workDirectory, fmt.Sprintf("%s_run.sh", executionConf.scriptFileName))

	executionConf.EnvInputFileName = envInputFileName
	executionConf.EntryScriptFileName = entryScriptFileName
	executionConf.EnvOutFileName = envOutFileName

	fmt.Println(entryScriptFileName, envOutFileName)
	err := godotenv.Write(executionConf.EnvInputVars, envInputFileName)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	entryScript, err := buildDockerEntryScript(executionConf.command, executionConf.args, executionConf.OutputVars, executionConf.EnvOutFileName)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	err = os.WriteFile(executionConf.EntryScriptFileName, []byte(entryScript), 0644) //TODO check mode with entry script
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	err = os.WriteFile(executionConf.EnvOutFileName, []byte(""), 0644) //TODO check mode with entry script
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	dockerRunCommand, err := buildDockerRunCommand(executionConf)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	fmt.Println(dockerRunCommand)
	//dockerRunCommand = "echo hello------;sleep 10; echo done------"
	err = os.WriteFile(executionConf.RunCommandFileName, []byte(dockerRunCommand), 0644)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	// docker run -it -v   -environment file  -p
	runScriptCMD := exec.Command("/bin/sh", executionConf.RunCommandFileName)
	//runScriptCMD.Env = inputEnvironmentVariable
	err = RunCommand(runScriptCMD)
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

func buildDockerEntryScript(command string, args []string, outputVars []string, envOutFileName string) (string, error) {
	entryTemplate := `#!/bin/sh
set -e
set -o pipefail
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
	finalScript, err := Tprintf(entryTemplate, templateData)
	if err != nil {
		return "", err
	}
	return finalScript, nil
}

func buildDockerRunCommand(executionConf *executionConf) (string, error) {
	cmdTemplate := `docker run \
--env-file {{.EnvInputFileName}} \
-v {{.EntryScriptFileName}}:/devtron_script/_entry.sh \
-v {{.EnvOutFileName}}:/devtron_script/_out.env \
{{- if .MountCode }}
-v {{.SourceCodeLocation}}:{{.SourceCodeMountLocation}} \
{{- end }}
{{ if .ScriptLocation -}}
-v {{ .ScriptLocation}}:{{.ScriptMountLocation}} \
{{- end}}
{{ range $hostPort, $ContainerPort := .ExposedPorts -}}
-p {{$hostPort}}:{{$ContainerPort}} \
{{ end }}
{{- .DockerImage}} \
/bin/sh /devtron_script/_entry.sh
`
	finalScript, err := Tprintf(cmdTemplate, executionConf)
	if err != nil {
		return "", err
	}
	return finalScript, nil

}
