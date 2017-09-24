package tpl

import (
	"os"
	"fmt"
	"runtime"
	"gopkg.in/yaml.v2"
)

type PlaceholderTemplate struct {
	template []byte
}

func NewPlaceholderTemplate(template []byte) (Template, error) {
	return PlaceholderTemplate{template}, nil
}

func (t PlaceholderTemplate) Render(param map[string]interface{}) (res []byte, err error) {
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
	err = yaml.Unmarshal(t.template, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	r, err := envsubst(string(t.template), param)
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
		if !ok {
			panic(fmt.Errorf("\"%s\" is not defined", key))
		}
		// https://github.com/ghodss/yaml/blob/master/yaml.go#L130
		/*
		switch v := value.(type) {
		case string: return v
		...
		}
		*/
		return fmt.Sprintf("%v", value)
	})
	return
}

