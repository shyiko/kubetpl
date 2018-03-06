package main

import (
	"io/ioutil"
	"testing"
)

func TestRender(t *testing.T) {
	config := map[string]interface{}{
		"NAME":    "nm",
		"MESSAGE": "msg",
	}
	renderedSh, err := render([]string{"example/nginx.shell.yml"}, config, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(renderedSh) == 0 {
		t.Fatal("len(rendered) == 0")
	}
	renderedGo, err := render([]string{"example/nginx.go-template.yml"}, config, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	renderedTk, err := render([]string{"example/nginx.template-kind.yml"}, config, "", "", false)
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

func TestRenderWithDataFromFile(t *testing.T) {
	config := map[string]interface{}{
		"NAME": "app",
	}
	actual, err := render([]string{"example/nginx-with-data-from-file.yml"}, config, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := ioutil.ReadFile("example/nginx-with-data-from-file.rendered.yml")
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}
