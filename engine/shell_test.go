package engine

import (
	log "github.com/Sirupsen/logrus"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestShellTemplateRender(t *testing.T) {
	actual, err := ShellTemplate{
		content: []byte(`# kubetpl:syntax:$
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: $NAME-deployment
  annotations:
    replicas-as-string: "$REPLICAS"
    key: "${NAME}$$VALUE" # $${...} and $$$$ test
spec:
  replicas: $REPLICAS
`),
		ignoreUnset: false,
	}.Render(map[string]interface{}{
		"NAME":     "app",
		"NOT_USED": "value",
		"REPLICAS": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `# kubetpl:syntax:$
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: app-deployment
  annotations:
    replicas-as-string: "1"
    key: "app$VALUE" # ${...} and $$ test
spec:
  replicas: 1
`
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", actual, expected)
	}
}

func TestShellTemplateRenderIncomplete(t *testing.T) {
	_, err := ShellTemplate{
		content: []byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: $NAME-deployment
`),
		ignoreUnset: false,
	}.Render(map[string]interface{}{
		"NOT_USED": "value",
	})
	if err == nil {
		t.Fatal()
	}
	expected := `4:9: "NAME" isn't set`
	if err.Error() != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", err.Error(), expected)
	}
}

func TestShellTemplateRenderUnresolved(t *testing.T) {
	actual, err := ShellTemplate{
		content: []byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: $NAME-deployment
  annotations:
    replicas-as-string: "${REPLICAS}" # $ $$ test
spec:
  replicas: $REPLICAS
`),
		ignoreUnset: true,
	}.Render(map[string]interface{}{
		"NAME": "app",
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: app-deployment
  annotations:
    replicas-as-string: "${REPLICAS}" # $ $ test
spec:
  replicas: $REPLICAS
`
	if string(actual) != expected {
		t.Fatalf("actual: \n%s != expected: \n%s", string(actual), expected)
	}
}
