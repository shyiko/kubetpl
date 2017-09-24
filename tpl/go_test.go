package tpl

import "testing"

func TestGoTemplateRender(t *testing.T) {
	actual, err := GoTemplate{
		[]byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: {{ .NAME }}-deployment
  annotations:
    replicas-as-string: {{ .REPLICAS | quote }}
spec:
  replicas: {{ .REPLICAS }}
`),
	}.Render(map[string]interface{}{
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
		t.Fatalf("actual: %#v != expected: %#v", actual, expected)
	}
}

func TestGoTemplateRenderIncomplete(t *testing.T) {
	_, err := GoTemplate{
		[]byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: {{ .NAME }}-deployment
`),
	}.Render(map[string]interface{}{
		"NOT_USED": "value",
	})
	if err == nil {
		t.Fatal()
	}
}
