package main

import (
	"testing"
)

func TestRender(t *testing.T) {
	config := map[string]interface{}{
		"NAME":    "nm",
		"MESSAGE": "msg",
	}
	renderedSh, err := render([]string{"example/nginx.shell.yml"}, config, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(renderedSh) == 0 {
		t.Fatal("len(rendered) == 0")
	}
	renderedGo, err := render([]string{"example/nginx.go-template.yml"}, config, "")
	if err != nil {
		t.Fatal(err)
	}
	renderedTk, err := render([]string{"example/nginx.template-kind.yml"}, config, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(renderedSh) != string(renderedGo) {
		t.Fatalf("sh: \n%s != go: \n%s", string(renderedSh), string(renderedGo))
	}
	if string(renderedGo) != string(renderedTk) {
		t.Fatalf("go: \n%s != tk: \n%s", string(renderedGo), string(renderedTk))
	}
}
