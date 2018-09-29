package engine

import (
	log "github.com/Sirupsen/logrus"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestGoTemplateRender(t *testing.T) {
	actual, err := Must(NewGoTemplate(
		[]byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: {{ .NAME }}-deployment
  annotations:
    replicas-as-string: {{ .REPLICAS | quote }}
spec:
  replicas: {{ .REPLICAS }}
`), "template"),
	).Render(map[string]interface{}{
		"NAME":     "app",
		"NOT_USED": "value",
		"REPLICAS": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: app-deployment
  annotations:
    replicas-as-string: "1"
spec:
  replicas: 1
`
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}

func TestGoTemplateRenderIncomplete(t *testing.T) {
	_, err := Must(NewGoTemplate(
		[]byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: {{ .NAME }}-deployment
`), "template"),
	).Render(map[string]interface{}{
		"NOT_USED": "value",
	})
	if err == nil {
		t.Fatal()
	}
}

func TestGoTemplateRenderIsSetGet(t *testing.T) {
	tpl := Must(NewGoTemplate(
		[]byte(`x{{ .VAR }}x
x{{ if isset "OPT_VAR" }}{{ .OPT_VAR }}{{ else }}default value{{ end }}x
x{{ if def . "OPT_VAR" }}{{ .OPT_VAR }}{{ else }}default value{{ end }}x
x{{ get "OPT_VAR" "default value" }}x
`), "template"),
	)
	actual, err := tpl.Render(map[string]interface{}{"VAR": "value", "OPT_VAR": "value"})
	if err != nil {
		t.Fatal(err)
	}
	expected := `xvaluex
xvaluex
xvaluex
xvaluex
`
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
	actualDef, err := tpl.Render(map[string]interface{}{"VAR": "value"})
	if err != nil {
		t.Fatal(err)
	}
	expectedDef := `xvaluex
xdefault valuex
xdefault valuex
xdefault valuex
`
	if string(actualDef) != expectedDef {
		t.Fatalf("actual: \n%s != expected: \n%s", actualDef, expectedDef)
	}
}
