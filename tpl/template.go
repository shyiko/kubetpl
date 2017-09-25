package tpl

type Template interface {
	Render(param map[string]interface{}) ([]byte, error)
}

func Must(t Template, err error) Template {
	if err != nil {
		panic(err)
	}
	return t
}
