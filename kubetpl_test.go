package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	cfg := map[string]interface{}{
		"NAME":    "nm",
		"MESSAGE": "msg",
	}
	opts := renderOpts{}
	renderedSh, err := render([]string{"example/nginx.$.yml"}, cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(renderedSh) == 0 {
		t.Fatal("len(rendered) == 0")
	}
	renderedGo, err := render([]string{"example/nginx.go-template.yml"}, cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	renderedTk, err := render([]string{"example/nginx.template-kind.yml"}, cfg, opts)
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

func TestRenderDirectory(t *testing.T) {
	cfg := map[string]interface{}{
		"NAME":    "nm",
		"MESSAGE": "msg",
	}
	opts := renderOpts{}
	renderedSh, err := render([]string{"example/directory"}, cfg, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(renderedSh) == 0 {
		t.Fatal("len(rendered) == 0")
	}
}

func TestRenderWithDataFromFile(t *testing.T) {
	// todo: test secret ("data" must be base64-encoded)
	src := []string{"example/nginx-with-data-from-file.yml"}
	cfg := map[string]interface{}{
		"NAME": "app",
	}
	if _, err := render(src, cfg, renderOpts{}); err == nil {
		t.FailNow()
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := render(src, cfg, renderOpts{chroot: "vendor"}); err == nil {
		t.FailNow()
	}
	if _, err := render(src, cfg, renderOpts{chroot: filepath.Join(cwd, "vendor")}); err == nil {
		t.FailNow()
	}
	if _, err := render(src, cfg, renderOpts{chroot: "example"}); err != nil {
		t.Fatal(err)
	}
	if _, err := render(src, cfg, renderOpts{chroot: filepath.Join(cwd, "example")}); err != nil {
		t.Fatal(err)
	}
	actual, err := render(src, cfg, renderOpts{chrootTemplateDir: true})
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

func TestFreeze(t *testing.T) {
	config := map[string]interface{}{
		"NAME":    "app",
		"MESSAGE": "msg",
	}
	actual, err := render([]string{"example/nginx-with-data-from-file.yml"}, config,
		renderOpts{freeze: true, chrootTemplateDir: true})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := ioutil.ReadFile("example/nginx-with-data-from-file.rendered+frozen.yml")
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}

func TestFreezeExternal(t *testing.T) {
	tmplFile, err := ioutil.TempFile("", "kubetpl-test")
	if err != nil {
		t.Fatal(err)
	}
	src := `
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  volumes:
  - name: app-volume
    configMap:
      name: app
`
	if err := ioutil.WriteFile(tmplFile.Name(), []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	config := map[string]interface{}{
		"NAME":    "app",
		"MESSAGE": "msg",
	}
	actual, err := render([]string{tmplFile.Name()}, config, renderOpts{
		freezeRefs: []string{"example/nginx.$.yml"},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `---
apiVersion: v1
kind: Pod
metadata:
  name: app
spec:
  volumes:
  - configMap:
      name: app-984c62c
    name: app-volume
`
	if string(actual) != string(expected) {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}

func TestFreezeIsExecutedLast(t *testing.T) {
	dataFileDir, err := ioutil.TempDir("", "kubetpl-test")
	if err != nil {
		t.Fatal(err)
	}
	dataFile := filepath.Join(dataFileDir, "file.txt")
	if err := ioutil.WriteFile(dataFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	tmplFile, err := ioutil.TempFile("", "kubetpl-test")
	if err != nil {
		t.Fatal(err)
	}
	src := `
apiVersion: v1
kind: ConfigMap
data:
  key: value
metadata:
  name: app
kubetpl/data-from-file: DATA_FILE
`
	if err := ioutil.WriteFile(tmplFile.Name(), []byte(strings.Replace(src, "DATA_FILE", dataFile, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	assertRenderedAs := func(expected string) {
		actual, err := render([]string{tmplFile.Name()}, nil, renderOpts{freeze: true, chrootTemplateDir: true})
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != string(expected) {
			t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
		}
	}
	assertRenderedAs(`---
apiVersion: v1
data:
  file.txt: ""
  key: value
kind: ConfigMap
metadata:
  name: app-fecea3a
`)
	if err := ioutil.WriteFile(dataFile, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	assertRenderedAs(`---
apiVersion: v1
data:
  file.txt: data
  key: value
kind: ConfigMap
metadata:
  name: app-caed0c7
`)
}

func TestCRLFIsNormalizedToLF(t *testing.T) {
	tmplFile, err := ioutil.TempFile("", "kubetpl-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(tmplFile.Name(), []byte("# kubetpl:syntax:$\r\nkind: ConfigMap\r\nmetadata:\r\n  name: $NAME"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]interface{}{
		"NAME": "windows",
	}
	actual, err := render([]string{tmplFile.Name()}, cfg, renderOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) == 0 {
		t.Fatal("len(rendered) == 0")
	}
	expected := "---\nkind: ConfigMap\nmetadata:\n  name: windows\n"
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}
