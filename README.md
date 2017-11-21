# kubetpl

Simple yet flexible client-side templating for Kubernetes.

Features:
- **Template flavor of your choice**.  
  Start simple ([$VAR](#placeholder)). [Step up your game with go-template|s](#go-template) when (and if!) needed). 
   
  [placeholder](#placeholder) (aka `$VAR` / `${VAR}`) | [go-template](#go-template) (enriched with [sprig](http://masterminds.github.io/sprig/)) | [template-kind](#template-kind) (aka `kind: Template`) are available out-of-box.  
  We also accept PRs for other formats. 
- Support for **\*.env** (`<VAR>=<VAL>`) and **YAML** data (config) files.
- Fail-fast defaults   
(all variables are considered to be required and must be given a value (unless explicitly marked optional)).    

## Installation

#### macOS / Linux

```sh
curl -sSL https://github.com/shyiko/kubetpl/releases/download/0.1.0/kubetpl-0.1.0-$(
    bash -c '[[ $OSTYPE == darwin* ]] && echo darwin || echo linux'
  )-amd64 > kubetpl && chmod a+x kubetpl
    
# verify PGP signature (optional but RECOMMENDED)
curl -sSL https://github.com/shyiko/kubetpl/releases/download/0.1.0/kubetpl-0.1.0-$(
    bash -c '[[ $OSTYPE == darwin* ]] && echo darwin || echo linux'
  )-amd64.asc > kubetpl.asc
curl https://keybase.io/shyiko/pgp_keys.asc | gpg --import
gpg --verify kubetpl.asc
```  

#### Windows

Download binary from the "[release(s)](https://github.com/shyiko/kubetpl/releases)" page.

## Usage

```sh
# config files can be either in .env (effectively ini without sections)
$ cat staging.env

NAME=sample-app
REPLICAS=1

# ... or YAML 
$ cat staging.env.yml

NAME: sample-app
REPLICAS: 1

# ... and considering that JSON is a subset of YAML, JSON
$ cat staging.env.json

{"NAME": "sample-app", "REPLICAS": 1}

# to render "placeholder" (aka $VAR / ${VAR}) type of template
# (e.g. https://github.com/shyiko/kubetpl/blob/master/example/nginx.yml.kubetpl)
$ kubetpl render svc-and-deploy.yml.kubetpl -i staging.env -s KEY=VALUE
# same as above
$ kubetpl render svc-and-deploy.yml --type=placeholder -i staging.env -s KEY=VALUE
$ kubetpl render svc-and-deploy.yml -P -i staging.env -s KEY=VALUE

# to render "go-template" type of template
# (e.g. https://github.com/shyiko/kubetpl/blob/master/example/nginx.yml.kubetpl-go)
$ kubetpl render svc-and-deploy.yml.kubetpl-go -i staging.yml -s KEY=VALUE
# same as above
$ kubetpl render svc-and-deploy.yml --type=go-template -i staging.env -s KEY=VALUE
$ kubetpl render svc-and-deploy.yml -G -i staging.env -s KEY=VALUE

# to render "template-kind" (aka "kind: Template") type of template
# (e.g. https://github.com/shyiko/kubetpl/blob/master/example/nginx.yml)
$ kubetpl render svc-and-deploy.yml -i staging.yml -s KEY=VALUE
# same as above
$ kubetpl render svc-and-deploy.yml --type=template-kind -i staging.env -s KEY=VALUE
$ kubetpl render svc-and-deploy.yml -T -i staging.env -s KEY=VALUE

# to apply template just pipe it through kubectl    
$ kubetpl render svc-and-deploy.yml.kubetpl -i k8s/staging.env | 
  kubectl apply -f -

# you can also render remote template(s)
$ kubetpl render https://rawgit.com/shyiko/kubetpl/master/example/nginx.yml.kubetpl \
  -s NAME=kubetpl-example-nginx -s MESSAGE="hello $(whoami)" | 
  kubectl apply -f -
# same as
$ printf "NAME=kubetpl-example-nginx\nMESSAGE=hello $(whoami)" > default.env
$ kubetpl render https://rawgit.com/shyiko/kubetpl/master/example/nginx.yml.kubetpl -i default.env | 
  kubectl apply -f -
```

> (for more examples see below)

## Template flavors

### placeholder

> aka $VAR / ${VAR}  

This is the type of template where all instances of $VAR / ${VAR} are replaced with corresponding values. If, for some variable, no value
is given - an error will be raised. 

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
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml.kubetpl</summary>

```yaml  
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

`kubetpl render k8s/svc-and-deploy.yml.kubetpl -i k8s/staging.env -s REPLICAS=3` should then yield

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

> Template `--type` is automatically inferred as `placeholder` if filename ends with `.yaml.kubetpl` or `.yml.kubetpl`. 
You can also specify it like this: `kubetpl k8s/svc-and-deploy.yml --type=placeholder ...`.

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
  <summary>&lt;project_dir&gt;/k8s/staging.env.yml</summary>

```yaml
NAME: sample-app
REPLICAS: 1
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml.kubetpl-go</summary>

```yaml  
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

`kubetpl render k8s/svc-and-deploy.yml.kubetpl-go -i k8s/staging.env.yml -s REPLICAS=3` should then yield

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

> Template `--type` is automatically inferred as `go-template` if filename ends with `.yaml.kubetpl-go` or `.yml.kubetpl-go`. 
You can also specify it like this: `kubetpl k8s/svc-and-deploy.yml --type=go-template ...`.

### template-kind

> aka `kind: Template`. 

Structure of the template is described in [Templates + Parameterization proposal](https://github.com/fabric8io/kubernetes-model/blob/master/vendor/k8s.io/kubernetes/docs/proposals/templates.md).

##### Example

Let's say we have the following (click to expand):

<details>
  <summary>&lt;project_dir&gt;/k8s/staging.env.yml</summary>

```yaml
NAME: sample-app
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml</summary>

```yaml  
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

`kubetpl render k8s/svc-and-deploy.yml -i k8s/staging.env.yml -s REPLICAS=3` should then yield

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

> PREREQUISITE: [go1.8+](https://golang.org/dl/).

```sh
git clone https://github.com/shyiko/kubetpl $GOPATH/src/github.com/shyiko/kubetpl 
cd $GOPATH/src/github.com/shyiko/kubetpl
make fetch

go run kubetpl.go
```

## Legal

All code, unless specified otherwise, is licensed under the [MIT](https://opensource.org/licenses/MIT) license.  
Copyright (c) 2017 Stanley Shyiko.
