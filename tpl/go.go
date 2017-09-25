package tpl

import (
	"bytes"
	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"text/template"
)

type GoTemplate struct {
	template []byte
}

func NewGoTemplate(template []byte) (Template, error) {
	return GoTemplate{template}, nil
}

func (t GoTemplate) Render(param map[string]interface{}) (res []byte, err error) {
	tmpl, err := template.New("template"). // todo: template name
						Funcs(funcMap()).
						Option("missingkey=error").
						Parse(string(t.template))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, param)
	if err != nil {
		return nil, err
	}
	res = buf.Bytes()
	// todo: not strictly needed, validation should catch any error
	m := map[string]interface{}{}
	if err = yaml.Unmarshal([]byte(buf.String()), &m); err != nil {
		return
	}
	return
}

// https://github.com/kubernetes/helm/blob/dece57e0baa94abdba22c0e3ced0b6ea64a83afd/pkg/engine/engine.go
func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()
	delete(f, "env")
	delete(f, "expandenv")
	return f
}
