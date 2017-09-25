package tpl

import (
	log "github.com/Sirupsen/logrus"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestKindTemplateRender(t *testing.T) {
	actual, err := Must(NewKindTemplate(
		[]byte(`kind: Template
apiVersion: v1
metadata:
  name: template
objects:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    annotations:
      b: $((BOOL))
      i: $((INT))
      s: $((STRING))
    name: cm-$(NAME)-$(ID)
- apiVersion: apps/v1beta1
  kind: Deployment
  metadata:
    name: deploy-$(NAME)-$(ID)
  spec:
    replicas: $((REPLICAS))
parameters:
- name: NAME
  required: true
  parameterType: string
- name: REPLICAS
  description: Number of replicas
  value: 1
  required: true
  parameterType: int
- name: STRING
  parameterType: string
- name: BOOL
  parameterType: int
- name: INT
  parameterType: int
`)),
	).Render(map[string]interface{}{
		"NAME":     "app",
		"NOT_USED": "value",
		"ID":       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `---
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    b: null
    i: null
    s: null
  name: cm-app-0
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: deploy-app-0
spec:
  replicas: 1
`
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}

func TestKindTemplateRenderIncomplete(t *testing.T) {
	_, err := Must(NewKindTemplate(
		[]byte(`kind: Template
apiVersion: v1
metadata:
  name: template
objects:
- apiVersion: v1
  kind: Pod
  metadata:
    name: pod-$(NAME)
parameters:
- name: NAME
  required: true
  parameterType: string
`),
	)).Render(map[string]interface{}{
		"NOT_USED": "value",
	})
	if err == nil || err.Error() != `"NAME" is missing` {
		t.Fatal()
	}
}
