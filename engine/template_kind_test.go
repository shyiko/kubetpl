package engine

import (
	log "github.com/sirupsen/logrus"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestKindTemplateRender(t *testing.T) {
	actual, err := Must(NewTemplateKindTemplate(
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
  parameterType: bool
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
	_, err := Must(NewTemplateKindTemplate(
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
	if err == nil || err.Error() != `"NAME" isn't set` {
		t.Fatal()
	}
}

func TestKindTemplateRenderDropNull(t *testing.T) {
	actual, err := Must(NewTemplateKindTemplate(
		[]byte(`kind: Template
apiVersion: v1
metadata:
  name: template
objects:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    ext/map1:
      "0": $((P))
      a: a
      a1: $((P))
      b: b
      b0: $((P))
      c: c
      d: $((D))
      c0: $((P))
    ext/map2:
      a: $((P))
      b: $((P))
      c: $((P))
    ext/map3:
      k: null
    name: $(NAME)
    ext/arr1: [$((P)), "a", $((P)), "b", $((P)), "c", $((P))]
    ext/arr2: [$((P)), $((P)), $((P))]
    ext/arr3: [null]
parameters:
- name: NAME
  required: true
  parameterType: string
- name: P
  parameterType: string
- name: D
  parameterType: string
  value: d
`), TemplateKindTemplateDropNull()),
	).Render(map[string]interface{}{"NAME": "app", "NOT_USED": "value"})
	if err != nil {
		t.Fatal(err)
	}
	expected := `---
apiVersion: v1
kind: ConfigMap
metadata:
  ext/arr1:
  - a
  - b
  - c
  ext/arr2: []
  ext/arr3:
  - null
  ext/map1:
    a: a
    b: b
    c: c
    d: d
  ext/map2: {}
  ext/map3:
    k: null
  name: app
`
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}
