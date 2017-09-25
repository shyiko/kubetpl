package tpl

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"runtime"
	"strconv"
	"strings"
)

type KindTemplate struct {
	Kind         string
	Objects      []map[interface{}]interface{}
	Parameters   []KindTemplateParameter
	ObjectLabels map[string]string
}

type KindTemplateParameter struct {
	Name        string
	DisplayName string
	Description string
	Value       interface{}
	Required    bool
	Type        string // string, int, bool or base64
}

func NewKindTemplate(template []byte) (Template, error) {
	var tpl KindTemplate
	yaml.Unmarshal(template, &tpl)
	if tpl.Kind != "Template" {
		return nil, errors.New("Invalid template (kind != Template)")
	}
	return tpl, nil
}

func (t KindTemplate) Render(param map[string]interface{}) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	data, err := t.data(param)
	if err != nil {
		return nil, err
	}
	log.Debugf("data = %v", data)
	var buf bytes.Buffer
	for _, obj := range t.Objects {
		/*
			m, ok := obj.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Expected an object, instead got %s (object #%v)", reflect.TypeOf(obj).String(), i)
			}
		*/
		uobj := traverse(
			obj,
			func(value string) interface{} {
				var implicit bool
				var null bool
				uvalue := expand(value, func(name string) string {
					if strings.HasPrefix(name, "(") && strings.HasSuffix(name, ")") {
						implicit = true
						name = name[1 : len(name)-1]
					}
					v, ok := data[name]
					if !ok {
						panic(fmt.Errorf("\"%s\" is not defined", name))
					}
					if v == nil {
						null = true
					}
					// fixme: make sure v is primitive
					return fmt.Sprintf("%v", v)
				})
				if null {
					return nil
				}
				if value != uvalue {
					if implicit {
						// https://github.com/ghodss/yaml/blob/master/yaml.go#L130
						switch strings.ToLower(uvalue) {
						case "true":
							return true
						case "false":
							return false
						}
						if v, err := strconv.ParseInt(uvalue, 0, 64); err == nil {
							return v
						}
						if v, err := strconv.ParseFloat(uvalue, 64); err == nil {
							return v
						}
					}
					return uvalue
				}
				return value
			},
		)
		b, err := yaml.Marshal(uobj)
		if err != nil {
			return nil, err
		}
		buf.Write([]byte("---\n"))
		buf.Write(b)
	}
	return buf.Bytes(), nil
}

func (t KindTemplate) data(param map[string]interface{}) (map[string]interface{}, error) {
	m := make(map[string]interface{}, len(param))
	for _, p := range t.Parameters {
		if param[p.Name] == nil {
			if p.Required && p.Value == nil {
				return nil, fmt.Errorf("\"%s\" is missing", p.Name)
			}
			m[p.Name] = p.Value
		}
	}
	for k, v := range param {
		m[k] = v
	}
	return m, nil
}

func traverse(m map[interface{}]interface{}, cb func(value string) interface{}) map[interface{}]interface{} {
	um := make(map[interface{}]interface{}, len(m)) // todo: no need to create a map if values are not updated
	for key, value := range m {
		switch v := value.(type) {
		case map[interface{}]interface{}:
			value = traverse(v, cb)
		case []interface{}:
			value = traverseSlice(v, cb)
		case string:
			value = cb(v)
		}
		um[key] = value
	}
	return um
}

func traverseSlice(m []interface{}, cb func(value string) interface{}) []interface{} {
	up := make([]interface{}, len(m)) // todo: no need to create a slice if values are not updated
	for index, value := range m {
		switch v := value.(type) {
		case map[interface{}]interface{}:
			value = traverse(v, cb)
		case []interface{}:
			value = traverseSlice(v, cb)
		case string:
			value = cb(v)
		}
		up[index] = value
	}
	return up
}

func expand(s string, mapping func(string) string) string {
	buf := make([]byte, 0, 2*len(s))
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			buf = append(buf, s[i:j]...)
			name, w := extractName(s[j:])
			buf = append(buf, mapping(name)...)
			j += w
			i = j
		}
	}
	if i < len(s) {
		buf = append(buf, s[i:]...)
	}
	return string(buf)
}

func extractName(s string) (string, int) {
	if s[1] == '(' {
		eq := 0
		for i := 2; i < len(s); i++ {
			if s[i] == '(' {
				eq++
			}
			if s[i] == ')' {
				if eq == 0 {
					return s[2:i], i + 1
				}
				eq--
			}
		}
	}
	return s, 0
}
