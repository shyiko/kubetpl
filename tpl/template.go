package tpl

type Template interface {
	Render(data map[string]interface{}) ([]byte, error)
}

func Must(t Template, err error) Template {
	if err != nil {
		panic(err)
	}
	return t
}

func isBasicYAMLType(v interface{}) bool {
	// https://github.com/ghodss/yaml/blob/master/yaml.go#L130
	switch v.(type) {
	case nil, string, bool, int, int64, float64, uint64:
		return true
	default:
		return false
	}
}
