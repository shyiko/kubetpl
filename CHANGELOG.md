# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [0.3.0](https://github.com/shyiko/kubetpl/compare/0.2.0...0.3.0) - 2018-04-16

### Added
- [ConfigMap/Secret freezing](https://github.com/shyiko/kubetpl#configmapsecret-freezing).
- [ConfigMap/Secret "data-from-file" injection](https://github.com/shyiko/kubetpl#configmapsecret-data-from-file-injection).
- `# kubetpl:syntax:<template flavor, e.g. $, go-template or template-kind>` directive (alleviates the need to specify `--syntax=<template flavor>` on the command line). e.g.

    ```yaml
    echo $'
    # kubetpl:syntax:$
    
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: $NAME
    ' | kubetpl render - -s NAME=example  
    ```
- [<kbd>Tab</kbd> completion](https://github.com/shyiko/kubetpl#tab-completion) (for bash and zsh).

### Deprecated
- `--syntax` hint through file extension (`*.kubetpl` for `$`, `*.kubetpl-go` for `go-template`). 
- `-P`, `-G`, `-T`, `-t/--type` CLI flags  
(use `--syntax` or `# kubetpl:syntax:<template flavor>` instead).

## [0.2.0](https://github.com/shyiko/kubetpl/compare/0.1.0...0.2.0) - 2018-01-15

### Added
- `$$` to represent literal dollar sign (`$$v` is interpreted as `$v` string and not `'$' + <value_of_v>`)  
(`$` template flavor).
- Location info (line/column) to error messages    
(`$` template flavor).

## 0.1.0 - 2017-09-26
