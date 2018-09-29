package engine

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"os"
	"text/template"
)

type GoTemplate struct {
	content []byte
	name    string
}

func NewGoTemplate(template []byte, name string) (Template, error) {
	return GoTemplate{template, name}, nil
}

func (t GoTemplate) Render(data map[string]interface{}) ([]byte, error) {
	tmpl, err := template.New(t.name).Funcs(funcMap(data)).Option("missingkey=error").Parse(string(t.content))
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
func funcMap(data map[string]interface{}) template.FuncMap {
	f := sprig.TxtFuncMap()
	f["isset"] = func(key string) interface{} {
		_, ok := data[key]
		return ok
	}
	f["get"] = func(key string, def interface{}) interface{} {
		if v, ok := data[key]; ok {
			return v
		}
		return def
	}
	f["def"] = func(m map[string]interface{}, key string) interface{} {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(`'{{ if def . "%s" }}' is deprecated (use '{{ if isset "%s" }}' instead)`, key, key))
		_, ok := m[key]
		return ok
	}
	delete(f, "env")
	delete(f, "expandenv")
	return f
}
