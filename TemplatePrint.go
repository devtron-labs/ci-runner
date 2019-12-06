package main

import (
	"bytes"
	"fmt"
	"text/template"
)

// Tprintf passed template string is formatted usign its operands and returns the resulting string.
// Spaces are added between operands when neither is a string.
func Tprintf(tmpl string, data interface{}) (string, error) {
	t := template.Must(template.New("tpl").Parse(tmpl))
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func main() {
	scriptTemplate := `#!/bin/sh
{{ range $key, $value := .envVr }}
export {{ $key }}={{ $value }} ;
{{ end }}
{{.script}};
`
	d := make(map[string]interface{})
	env := make(map[string]string)
	env["ID"] = "1"
	d["script"] = "ls -lrth"
	d["envVr"] = env
	t, err := Tprintf(scriptTemplate, d)
	fmt.Println(t)
	fmt.Println(err)
}
