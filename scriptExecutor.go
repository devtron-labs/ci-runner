package main

import (
	"bufio"
	"fmt"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const doubleQuoteSpecialChars = "\\\n\r\"!$`"

func RunScriptsV1(outputPath string, bashScript string, script string, envVars map[string]string) error {
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
	finalScript, err := Tprintf(scriptTemplate, templateData)
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
	err = util.RunCommand(runScriptCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func RunScripts(workDirectory string, scriptFileName string, script string, envInputVars map[string]string, outputVars []string) (map[string]string, error) {
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
	//add sysytem env variable
	for k, v := range getSystemEnvVariables() {
		//add only when not overriden by user
		if _, ok := envInputVars[k]; !ok {
			envInputVars[k] = v
		}
	}
	var inputEnvironmentVariable []string
	for k, v := range envInputVars {
		inputEnvironmentVariable = append(inputEnvironmentVariable, fmt.Sprintf("%s=%s", k, v))
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	runScriptCMD.Env = inputEnvironmentVariable
	err = util.RunCommand(runScriptCMD)
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
	finalScript, err := Tprintf(scriptTemplate, templateData)
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

func RunScriptsInDocker(executionConf *executionConf) (map[string]string, error) {
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

	// Remove double quotes at this point and then observe the behaviour, might be a possibility that double quotes are inserted before this point

	for key, value := range executionConf.EnvInputVars {
		log.Println("The key is ", key, " and the value is ", value)
	}

	err := Write(executionConf.EnvInputVars, envInputFileName)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}

	//file, err := os.Create(envInputFileName)
	//if err != nil {
	//	panic(err)
	//}
	//defer file.Close()
	//
	//// Create a buffered writer
	//writer := bufio.NewWriter(file)
	//
	//// Write each key-value pair to the file
	//for key, value := range executionConf.EnvInputVars {
	//	// Format the key-value pair
	//	log.Println("The key is ", key, " and the value is ", value)
	//	line := fmt.Sprintf(`%s : %s\n`, key, value)
	//	_, err = file.WriteString(line)
	//	if err != nil {
	//		panic(err)
	//	}
	//}
	//
	//err = writer.Flush()
	//if err != nil {
	//	panic(err)
	//}

	file, err := os.Open(envInputFileName)
	if err != nil {
		log.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	//Create a new scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	//Read the file line by line
	for scanner.Scan() {
		line := scanner.Text()
		log.Println("Line received is ", line)
	}

	//Check if there was an error while scanning
	if err = scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
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
	log.Println("docker run command is ", dockerRunCommand)
	//dockerRunCommand = "echo hello------;sleep 10; echo done------"
	err = os.WriteFile(executionConf.RunCommandFileName, []byte(dockerRunCommand), 0644)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return nil, err
	}
	// docker run -it -v   -environment file  -p
	runScriptCMD := exec.Command("/bin/sh", executionConf.RunCommandFileName)
	//runScriptCMD.Env = inputEnvironmentVariable
	err = util.RunCommand(runScriptCMD)
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
	log.Println("output variables are ", outputVars)
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
	finalScript, err := Tprintf(entryTemplate, templateData)
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
	finalScript, err := Tprintf(cmdTemplate, executionConf)
	if err != nil {
		return "", err
	}
	return finalScript, nil

}

func Write(envMap map[string]string, filename string) error {
	content, err := Marshal(envMap)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content + "\n")
	if err != nil {
		return err
	}
	file.Sync()
	return err
}

func Marshal(envMap map[string]string) (string, error) {
	lines := make([]string, 0, len(envMap))
	for k, v := range envMap {
		if d, err := strconv.Atoi(v); err == nil {
			lines = append(lines, fmt.Sprintf(`%s=%d`, k, d))
		} else {
			lines = append(lines, fmt.Sprintf(`%s=%s`, k, v))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

func doubleQuoteEscape(line string) string {
	for _, c := range doubleQuoteSpecialChars {
		toReplace := "\\" + string(c)
		if c == '\n' {
			toReplace = `\n`
		}
		if c == '\r' {
			toReplace = `\r`
		}
		line = strings.Replace(line, string(c), toReplace, -1)
	}
	return line
}
