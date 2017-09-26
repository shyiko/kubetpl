package tpl

import (
	"bytes"
	"github.com/Masterminds/sprig"
	"text/template"
)

type GoTemplate struct {
	content []byte
	name string
}

func NewGoTemplate(template []byte, name string) (Template, error) {
	return GoTemplate{template, name}, nil
}

func (t GoTemplate) Render(data map[string]interface{}) ([]byte, error) {
	tmpl, err := template.New(t.name).Funcs(funcMap()).Option("missingkey=error").Parse(string(t.content))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// https://github.com/kubernetes/helm/blob/dece57e0baa94abdba22c0e3ced0b6ea64a83afd/pkg/engine/engine.go
func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()
	delete(f, "env")
	delete(f, "expandenv")
	return f
}
