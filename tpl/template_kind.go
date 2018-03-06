package tpl

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	yamlext "github.com/shyiko/kubetpl/yaml"
	"gopkg.in/yaml.v2"
	"runtime"
	"strconv"
	"strings"
)

type mixedContentTemplate struct {
	doc []interface{}
}

type TemplateKindTemplate struct {
	Kind         string
	Objects      []map[interface{}]interface{}
	Parameters   []TemplateKindTemplateParameter
	ObjectLabels map[string]string // todo: not implemented
}

type TemplateKindTemplateParameter struct {
	Name        string
	DisplayName string
	Description string
	Value       interface{}
	Required    bool
	Type        string // string, int, bool or base64 (optional just like rest of the fields (except name))
}

func NewTemplateKindTemplate(template []byte) (Template, error) {
	var doc []interface{}
	for _, chunk := range yamlext.Chunk(template) {
		var tpl TemplateKindTemplate
		err := yaml.Unmarshal(chunk, &tpl)
		if err != nil {
			return nil, err
		}
		if tpl.Kind == "Template" {
			doc = append(doc, tpl)
			continue
		}
		// otherwise assume that it's a regular resource
		m := make(map[string]interface{})
		if err := yaml.Unmarshal(chunk, &m); err != nil {
			return nil, err
		}
		if len(m) == 0 {
			// empty doc
			continue
		}
		if m["kind"] == nil || m["kind"] == "" {
			return nil, errors.New("Resource \"kind\" is missing")
		}
		doc = append(doc, m)
	}
	return mixedContentTemplate{doc}, nil
}

func (t mixedContentTemplate) Render(data map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	for _, doc := range t.doc {
		var res []byte
		var err error
		switch d := doc.(type) {
		case TemplateKindTemplate:
			res, err = d.Render(data)
			if err != nil {
				return nil, err
			}
			buf.Write(res)
		default:
			res, err = yaml.Marshal(d)
			if err != nil {
				return nil, err
			}
			buf.Write([]byte("---\n"))
			buf.Write(res)
		}
	}
	return buf.Bytes(), nil
}

func (t TemplateKindTemplate) Render(data map[string]interface{}) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	data, err = t.data(data)
	if err != nil {
		return nil, err
	}
	log.Debugf("data = %v", data)
	var buf bytes.Buffer
	for _, obj := range t.Objects {
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
						panic(fmt.Errorf("\"%s\" isn't set", name))
					}
					if v == nil {
						null = true
					}
					if !yamlext.IsBasicType(value) {
						panic(fmt.Errorf("\"%s\" must be either a string, number or a boolean", name))
					}
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

func (t TemplateKindTemplate) data(param map[string]interface{}) (map[string]interface{}, error) {
	m := make(map[string]interface{}, len(param))
	for _, p := range t.Parameters {
		if param[p.Name] == nil {
			if p.Required && p.Value == nil {
				return nil, fmt.Errorf("\"%s\" isn't set", p.Name)
			}
			m[p.Name] = p.Value
		}
	}
	for k, v := range param {
		m[k] = v
	}
	// enforce p.Type
	for _, p := range t.Parameters {
		if p.Type == "" {
			continue // type is optional
		}
		switch p.Type {
		case "string", "base64", "int", "bool":
			break
		default:
			return nil, fmt.Errorf("\"parameterType\" of \"%s\" must be either string, base64, int or bool", p.Name)
		}
		v := m[p.Name]
		if !yamlext.IsBasicType(v) {
			return nil, fmt.Errorf("Type of \"%s\" must be \"%s\"", p.Name, p.Type)
		}
		switch p.Type {
		case "base64":
			if !isBase64EncodedString(v) {
				return nil, fmt.Errorf("\"%s\" must be a base64-encoded string", p.Name)
			}
		case "int":
			if !isInt(v) {
				return nil, fmt.Errorf("\"%s\" must be a number", p.Name)
			}
		case "bool":
			if !isBool(v) {
				return nil, fmt.Errorf("\"%s\" must be a boolean", p.Name)
			}
		}
	}
	return m, nil
}

func isBase64EncodedString(v interface{}) bool {
	_, err := base64.StdEncoding.DecodeString(fmt.Sprintf("%v", v))
	return err == nil
}

func isInt(v interface{}) bool {
	vs := fmt.Sprintf("%v", v)
	if _, err := strconv.ParseInt(vs, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(vs, 64); err == nil {
		return true
	}
	return false
}

func isBool(v interface{}) bool {
	switch strings.ToLower(fmt.Sprintf("%v", v)) {
	case "true", "false":
		return true
	}
	return false
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
