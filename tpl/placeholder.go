package tpl

import (
	"fmt"
	"gopkg.in/yaml.v2"
	yamlext "github.com/shyiko/kubetpl/yml"
	"os"
	"runtime"
)

type PlaceholderTemplate struct {
	content []byte
}

func NewPlaceholderTemplate(template []byte) (Template, error) {
	return PlaceholderTemplate{template}, nil
}

func (t PlaceholderTemplate) Render(data map[string]interface{}) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	// ensure that input is a valid yaml even if expansion is done over the whole string
	// and not individual nodes (for now)
	if err := yamlext.UnmarshalSlice(t.content, func(in []byte) error {
		return yaml.Unmarshal(in, map[string]interface{}{})
	}); err != nil {
		return nil, err
	}
	r, err := envsubst(string(t.content), data)
	if err != nil {
		return nil, err
	}
	return []byte(r), nil
}

func envsubst(value string, env map[string]interface{}) (res string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	res = os.Expand(value, func(key string) string {
		value, ok := env[key]
		// todo: handle nil same way as template_kind.go
		if !ok || value == nil {
			panic(fmt.Errorf("\"%s\" isn't set", key))
		}
		if !isBasicYAMLType(value) {
			panic(fmt.Errorf("\"%s\" must be either a string, number or a boolean", key))
		}
		return fmt.Sprintf("%v", value)
	})
	return
}
