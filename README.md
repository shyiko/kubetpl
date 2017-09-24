# kubetpl

Simple yet flexible client-side templating for Kubernetes.

Features:
- **Template flavour of your choice**.  
  Start simple ([$variable]()). [Step up your game with go-template|s]() when (and if!) needed). 
   
  [placeholder]() | [go-template]() are available out-of-box.  
  We also accept PRs for other formats. 
- Support for data (config) files in **YAML** and **.env** (`<key>=<value>`).   
- Built-in **client-side resource validation** (backed by [kubeval](https://github.com/garethr/kubeval)).

> for Secret management prefer [kubesec](https://github.com/shyiko/kubesec).

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

<details>
  <summary>&lt;project_dir&gt;/k8s/staging.env</summary>
```ini  
NAME=sample-app
REPLICAS=1
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml.ktpl</summary>
```yml  
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

```sh
$ kubetpl render k8s/svc-and-deploy.yml.ktpl -c k8s/staging.env -d REPLICAS=3
 
apiVersion: v1
kind: Service
metadata:
  name: sample-app
# omitted
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
# omitted
```

`format` is automatically inferred as `placeholder` if filename ends with `.yaml.ktpl` or `.yml.ktpl`. 

#### [go-template](https://golang.org/pkg/text/template/)

> All functions provided by [sprig](http://masterminds.github.io/sprig/) are available  
(with the exception of `env` and `expandenv`).

##### Example

<details>
  <summary>&lt;project_dir&gt;/k8s/staging.env.yml</summary>
```ini  
NAME: sample-app
REPLICAS: 1
```
</details>
<details>
  <summary>&lt;project_dir&gt;/k8s/svc-and-deploy.yml.goktpl</summary>
```yml  
apiVersion: v1
kind: Service
metadata:
  name: {{ .NAME }}
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

```sh
$ kubetpl render k8s/svc-and-deploy.yml.goktpl -c k8s/staging.env.yml -d REPLICAS=3
 
apiVersion: v1
kind: Service
metadata:
  name: sample-app
# omitted
---
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: sample-app-deployment
spec:
  replicas: 3
# omitted
```

NOTE: `format` is automatically inferred as `go-template` if filename ends with `.yaml.goktpl` or `.yml.goktpl`. 

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
