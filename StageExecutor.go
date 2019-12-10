package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)


func RunScripts(outputPath string, bashScript string, script string, envVars map[string]string) error {
	log.Println("running script commands")
	scriptTemplate := `#!/bin/sh
{{ range $key, $value := .envVr }}
export {{ $key }}={{ $value }} ;
{{ end }}
{{.script}}
`

	templateData := make(map[string]interface{})
	templateData["envVr"] = envVars
	templateData["script"] = script
	finalScript, err := Tprintf(scriptTemplate, templateData)
	if err != nil {
		log.Println(devtron, err)
		return err
	}
	err = os.MkdirAll(outputPath, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(devtron, err)
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
	log.Println(devtron, scriptPath)
	if err != nil {
		log.Println(devtron, err)
		return err
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	err = RunCommand(runScriptCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
