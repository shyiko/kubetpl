# kubetpl

Simple yet flexible client-side templating for Kubernetes.

Features:
- **Template flavour of your choice**.  
  Start simple ([$variable](#placeholder)). [Step up your game with go-template|s](#go-template) when (and if!) needed). 
   
  [placeholder](#placeholder) | [go-template](#go-template) | [kind-eq-template](#kind-eq-template) are available out-of-box.  
  We also accept PRs for other formats. 
- Support for data (config) files in **YAML** and **.env** (`<key>=<value>`).   
- Built-in **client-side resource validation** (backed by [kubeval](https://github.com/garethr/kubeval)).

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
kubetpl render k8s/svc-and-deploy.yml.ktpl -c k8s/staging.env -d KEY=<command_line_override> | 
  kubectl apply -f -

kubetpl render https://rawgit.com/shyiko/kubetpl/master/example/nginx.yml.ktpl \
  -d NAME=kubetpl-example-nginx -d MESSAGE="hello $(whoami)" | 
  kubectl apply -f -
# same as
printf "NAME=kubetpl-example-nginx\nMESSAGE=hello $(whoami)" > default.env
kubetpl render https://rawgit.com/shyiko/kubetpl/master/example/nginx.yml.ktpl -c default.env | 
  kubectl apply -f -
  
```

### Template flavours

#### placeholder

> aka $variable / ${variable}  

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
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml.ktpl</summary>

```yaml  
apiVersion: v1
kind: Service
metadata:
  name: $NAME
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

Executing `kubetpl render k8s/svc-and-deploy.yml.ktpl -c k8s/staging.env -d REPLICAS=3` should yield

```yaml
# omitted
metadata:
  name: sample-app-service
spec:
  selector:
    app: sample-app
# omitted
---
# omitted
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
  template: 
    metadata:
      labels:
        app: sample-app
# omitted
```

> `--format` is automatically inferred as `placeholder` if filename ends with `.yaml.ktpl` or `.yml.ktpl`. 

#### [go-template](https://golang.org/pkg/text/template/)

> All functions provided by [sprig](http://masterminds.github.io/sprig/) are available  
(with the exception of `env` and `expandenv`).

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
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml.goktpl</summary>

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

Executing `kubetpl render k8s/svc-and-deploy.yml.goktpl -c k8s/staging.env.yml -d REPLICAS=3` should yield

```yaml
# omitted
metadata:
  name: sample-app-service
spec:
  selector:
    app: sample-app
# omitted
---
# omitted
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
  template: 
    metadata:
      labels:
        app: sample-app
# omitted
```

> `--format` is automatically inferred as `go-template` if filename ends with `.yaml.goktpl` or `.yml.goktpl`. 

#### kind-eq-template

> aka [kind=Template](https://github.com/fabric8io/kubernetes-model/blob/master/vendor/k8s.io/kubernetes/docs/proposals/templates.md). 

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
labels:
  template: nginx-template
objects:
- apiVersion: v1
  kind: Service
  metadata:
    name: $(NAME)
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

Executing `kubetpl render k8s/svc-and-deploy.yml -c k8s/staging.env.yml -d REPLICAS=3` should yield

```yaml
# omitted
metadata:
  name: sample-app-service
spec:
  selector:
    app: sample-app
# omitted
---
# omitted
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
  template: 
    metadata:
      labels:
        app: sample-app
# omitted
```

## Development

> PREREQUISITE: [go1.8](https://golang.org/dl/).

```sh
git clone https://github.com/shyiko/kubetpl $GOPATH/src/github.com/shyiko/kubetpl 
cd $GOPATH/src/github.com/shyiko/kubetpl
make fetch

go run kubetpl.go
```

## Legal

All code, unless specified otherwise, is licensed under the [MIT](https://opensource.org/licenses/MIT) license.  
Copyright (c) 2017 Stanley Shyiko.
