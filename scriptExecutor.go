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
	// system generate values
	EnvInputFileName    string // system generated
	EntryScriptFileName string // system generated
}

func RunScriptsInDocker(workDirectory string, scriptFileName string, executionConf *executionConf) (map[string]string, error) {
	envInputFileName := filepath.Join(workDirectory, fmt.Sprintf("%s_in.env", scriptFileName))
	entryScriptFileName := filepath.Join(workDirectory, fmt.Sprintf("%s_entry.sh", scriptFileName))
	envOutFileName := filepath.Join(workDirectory, fmt.Sprintf("%s_in.env", scriptFileName))

	fmt.Println(entryScriptFileName, envOutFileName)
	err := godotenv.Write(executionConf.EnvInputVars, envInputFileName)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}

	// docker run -it -v   -environment file  -p

	return nil, nil
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
	templateData["envOutFileName"] = envOutFileName
	templateData["outputVars"] = outputVars
	finalScript, err := Tprintf(entryTemplate, templateData)
	if err != nil {
		return "", err
	}
	return finalScript, nil
}

func buildDockerRunCommand(executionConf *executionConf) (string, error) {
	cmdTemplate := `docker run -it \
--env-file {{.EnvInputFileName}} \
-v {{.EntryScriptFileName}}:/devtron_script/_entry.sh \
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
