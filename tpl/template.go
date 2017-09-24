package tpl

type Template interface {
	Render(param map[string]interface{}) ([]byte, error)
}
