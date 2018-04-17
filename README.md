# kubetpl ![Latest Version](https://img.shields.io/badge/latest-0.3.0-blue.svg) [![Build Status](https://travis-ci.org/shyiko/kubetpl.svg?branch=master)](https://travis-ci.org/shyiko/kubetpl)

Kubernetes templates made easy.  
\#keep-it-simple \#no-server-component

Features:
- **Template flavor of your choice**
  - [$](#$) (`$VAR` / `${VAR}`);
  - [go-template](#go-go-template) (go-template enriched with [sprig](http://masterminds.github.io/sprig/)); 
  - [template-kind](#template-kind) (`kind: Template`).
- Support for **\*.env** (`<VAR>=<VAL>`) and **YAML** / **JSON** config files.
- **Fail-fast defaults** (all variables must be given a value (unless explicitly marked optional)).
- [ConfigMap/Secret freezing](#configmapsecret-freezing) for easier and less error-prone ConfigMap/Secret rollouts  
(something to consider ~~if~~ when you hit [kubernetes/kubernetes#22368](https://github.com/kubernetes/kubernetes/issues/22368)).  
- [ConfigMap/Secret "data-from-file" injection](#configmapsecret-data-from-file-injection) when `kubectl create configmap ... --from-file=... --from-file=... --from-file=... ...` feels like too much typing.
- `image:tag` -> `image@digest` pinning with the help of [dockry](https://github.com/shyiko/dockry)   
(e.g. `kubetpl render -s IMAGE=$(dockry digest --fq user/image:master) ...` to force redeployment of the new build published under the same tag).

## Installation

#### macOS / Linux

```sh
curl -sSL https://github.com/shyiko/kubetpl/releases/download/0.3.0/kubetpl-0.3.0-$(
    bash -c '[[ $OSTYPE == darwin* ]] && echo darwin || echo linux'
  )-amd64 -o kubetpl && chmod a+x kubetpl && sudo mv kubetpl /usr/local/bin/
```
    
Verify PGP signature (optional but recommended):    
```sh
curl -sSL https://github.com/shyiko/kubetpl/releases/download/0.3.0/kubetpl-0.3.0-$(
    bash -c '[[ $OSTYPE == darwin* ]] && echo darwin || echo linux'
  )-amd64.asc -o kubetpl.asc
curl -sS https://keybase.io/shyiko/pgp_keys.asc | gpg --import
gpg --verify kubetpl.asc /usr/local/bin/kubetpl
```  

> macOS: `gpg` can be installed with `brew install gnupg`

#### Windows

Download executable from the [Releases](https://github.com/shyiko/kubetpl/releases) page.

## Usage

```sh
# create template
echo $'
# kubetpl:syntax:$

apiVersion: v1
kind: Pod
metadata:
  name: $NAME-pod
spec:
  containers:
  - name: $NAME-container
    image: $IMAGE
    env:
    - name: ENV_KEY
      value: $ENV_KEY
' > template.yml 

# create config file (.env, .yml/.yaml or .json) (optional)
# (you'll probably have a different config file for each cluster/namespace/etc)
echo $'
NAME=sample-app
ENV_KEY=value
' > staging.env
# you might not need a config file if there are only a handful of variables (like in this case)
# -s/--set key=value might be enough

# render template
kubetpl render template.yml -i staging.env -s IMAGE=nginx 

# to apply, pipe "render"ed output through kubectl    
kubetpl render template.yml -i staging.env -s IMAGE=nginx | 
  kubectl apply -f -
  
# you can also apply remote template(s) 
kubetpl render https://rawgit.com/shyiko/kubetpl/master/example/nginx.sh.yml \
  -s NAME=kubetpl-example-nginx -s MESSAGE="hello $(whoami)" | 
  kubectl apply -f -
```

> (for more examples see [Template flavors](#template-flavors))

#### <kbd>Tab</kbd> completion

```sh
# bash
source <(kubetpl completion bash)

# zsh
source <(kubetpl completion zsh)
```

## ConfigMap/Secret freezing

When `kubetpl render --freeze ...` is used, kubetpl rewrites `ConfigMap`/`Secret`'s name to include hash of the content 
and then updates all the references (in `Pod`s / `DaemonSet`s / `Deployment`s / `Job`s / `ReplicaSet`s / `ReplicationController`s / `StatefulSet`s / `CronJob`s) with a new value.

For example, executing [`kubetpl render --freeze example/nginx-with-configmap-frozen.sh.yml -s NAME=app -s MESSAGE=msg`](example/nginx-with-configmap-frozen.sh.yml) 
should produce [example/nginx-with-data-from-file.rendered+frozen.yml](example/nginx-with-data-from-file.rendered+frozen.yml).
 
NOTE: this feature can be used regardless of the [Template flavor](#template-flavors) choice (or lack thereof (i.e. on its own)).

## ConfigMap/Secret "data-from-file" injection

Optionally, ConfigMap/Secret|s can be extended with `kubetpl/data-from-file` to load "data" from a list of files (relative to a template unless a different `-c/--chroot` is specified), e.g.  

```yaml
kind: ConfigMap
kubetpl/data-from-file: 
  - file 
  - path/to/another-file
  - custom-key=yet-another-file
data:
  key: value
...
``` 

Upon `kubetpl render` the content of `file`, `another-file` and `yet-another-file` (using `custom-key` as a key)
will be added to the object's "data" (`kubetpl/data-from-file` is automatically striped away).

For example, executing [`kubetpl render --allow-fs-access example/nginx-with-data-from-file.yml -s NAME=app`](example/nginx-with-data-from-file.yml) 
should produce [example/nginx-with-data-from-file.rendered.yml](example/nginx-with-data-from-file.rendered.yml).

NOTE #1: for security reasons, `kubetpl/data-form-file` is not allowed to read files unless `--allow-fs-access` or `-c/--chroot=<root dir>` is specified (see `kubetpl render --help` for more). 

NOTE #2: this feature can be used regardless of the [Template flavor](#template-flavors) choice (or lack thereof (i.e. on its own)).

## Template flavors

Template syntax is determined by first checking template for `# kubetpl:syntax:<$|go-template|template-kind>` comment 
and then, if not found, `--syntax=<$|go-template|template-kind>` command line option. In the absence of both, 
kubetpl assumes that template is a regular resource definition file.

### $

A type of template where all instances of $VAR / ${VAR} are replaced with corresponding values. If, for some variable, no value
is given - an error will be raised. 

> Use `$$` when you need a literal dollar sign (`$$v` is interpreted as `$v` string and not `'$' + <value_of_v>`). 

##### Example

Let's say we have the following (click to expand):

<details>
  <summary>&lt;project_dir&gt;/k8s/staging.env</summary>

```ini  
NAME=sample-app
REPLICAS=1
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/template.yml</summary>

```yaml  
# kubetpl:syntax:$

apiVersion: v1
kind: Service
metadata:
  name: $NAME-service
spec:
  selector:
    app: $NAME
  ports:
  - protocol: TCP
    port: 80
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: $NAME-deployment
spec:
  replicas: $REPLICAS
  template: 
    metadata:
      labels:
        app: $NAME
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
```
</details>
<p><p>

`kubetpl render k8s/template.yml -i k8s/staging.env -s REPLICAS=3` should then yield

<details>
  <summary>(click to expand)</summary>

```yaml
apiVersion: v1
kind: Service
metadata:
  name: sample-app-service
spec:
  selector:
    app: sample-app
  ports:
  - protocol: TCP
    port: 80
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
  template: 
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
```
</details>
<p><p>

### go-template

> All functions provided by [sprig](http://masterminds.github.io/sprig/) are available  
(with the exception of `env` and `expandenv`).

A good overview of go-template|s can be found [here](https://gohugo.io/templates/introduction/#variables). You might also want to check [official documentation](https://golang.org/pkg/text/template/).

Some of the most commonly used expressions:
* `{{ .VAR }}` - get the value of `VAR`;
* `{{ .VAR | quote }}` - quote the value of VAR;   
* `{{ .VAR | indent 4 }}` - indent value of VAR with 4 spaces;   
* `{{ .VAR | b64enc }}` - base64-encode value of VAR;   
* `{{- if def . VAR }} ... {{- end }}` - render content between `}}` and `{{` only if .VAR is set.   

##### Example

Let's say we have the following (click to expand):

<details>
  <summary>&lt;project_dir&gt;/k8s/staging.env</summary>

```yaml
NAME=sample-app
REPLICAS=1
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/template.yml</summary>

```yaml
# kubetpl:syntax:go-template

apiVersion: v1
kind: Service
metadata:
  name: {{ .NAME }}-service
spec:
  selector:
    app: {{ .NAME }}
  ports:
  - protocol: TCP
    port: 80
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: {{ .NAME }}-deployment
spec:
  replicas: {{ .REPLICAS }}
  template: 
    metadata:
      labels:
        app: {{ .NAME }}
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
```
</details>
<p><p>

`kubetpl render k8s/template.yml -i k8s/staging.env -s REPLICAS=3` should then yield

<details>
  <summary>(click to expand)</summary>

```yaml
apiVersion: v1
kind: Service
metadata:
  name: sample-app-service
spec:
  selector:
    app: sample-app
  ports:
  - protocol: TCP
    port: 80
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
  template: 
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
```
</details>
<p><p>

### template-kind

> aka `kind: Template`. 

As described in [Templates + Parameterization proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/apps/OBSOLETE_templates.md).

##### Example

Let's say we have the following (click to expand):

<details>
  <summary>&lt;project_dir&gt;/k8s/staging.env</summary>

```yaml
NAME=sample-app
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/template.yml</summary>

```yaml
# kubetpl:syntax:template-kind

kind: Template
apiVersion: v1
metadata:
  name: nginx-template
  annotations:
    description: nginx template
objects:
- apiVersion: v1
  kind: Service
  metadata:
    name: $(NAME)-service
  spec:
    selector:
      app: $(NAME)
    ports:
    - protocol: TCP
      port: 80
- apiVersion: apps/v1beta1
  kind: Deployment
  metadata:
    name: $(NAME)-deployment
  spec:
    replicas: $((REPLICAS))
    template: 
      metadata:
        labels:
          app: $(NAME)
      spec:
        containers:
        - name: nginx
          image: nginx:1.7.9
          ports:
          - containerPort: 80
parameters:
- name: NAME
  description: Application name
  required: true
  parameterType: string
- name: REPLICAS
  description: Number of replicas
  value: 1
  required: true
  parameterType: int
```
</details>
<p><p>

`kubetpl render k8s/template.yml -i k8s/staging.env -s REPLICAS=3` should then yield

<details>
  <summary>(click to expand)</summary>

```yaml
apiVersion: v1
kind: Service
metadata:
  name: sample-app-service
spec:
  selector:
    app: sample-app
  ports:
  - protocol: TCP
    port: 80
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
  template: 
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
```
</details>

## Development

> PREREQUISITE: [go1.9+](https://golang.org/dl/).

```sh
git clone https://github.com/shyiko/kubetpl $GOPATH/src/github.com/shyiko/kubetpl 
cd $GOPATH/src/github.com/shyiko/kubetpl
make fetch

go run kubetpl.go
```

## Legal

All code, unless specified otherwise, is licensed under the [MIT](https://opensource.org/licenses/MIT) license.  
Copyright (c) 2018 Stanley Shyiko.
