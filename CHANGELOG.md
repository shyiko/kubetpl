# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [0.9.0](https://github.com/shyiko/kubetpl/compare/0.8.0...0.9.0) - 2019-01-16

### Added
- `--freeze`ing of `PodPreset`s.

### Fixed
- List of `--freeze` paths ([#12](https://github.com/shyiko/kubetpl/issues/12)).

## [0.8.0](https://github.com/shyiko/kubetpl/compare/0.7.1...0.8.0) - 2018-09-28

### Added
- `# kubetpl:set:KEY=VALUE` directive as a way to provide defaults on "per-template" basis, e.g.
    ```yaml
    echo $'
    # kubetpl:syntax:$
    # kubetpl:set:NAME=nginx
    # kubetpl:set:REPLICAS=1

    apiVersion: apps/v1beta1
    kind: Deployment
    metadata:
      name: $NAME
    spec:
      replicas: $REPLICAS
      template:
        metadata:
          labels:
            name: $NAME
        spec:
          containers:
          - name: nginx
            image: nginx:$VERSION
    ' | kubetpl render - -s VERSION=1.7.9
    ```
- (go-template) `{{ if isset "VAR" }}...{{ end }}` to check if `VAR` is set.
- (go-template) `{{ get "VAR" "default value" }}` as shorthand for `{{ if isset "VAR" }}{{ .VAR }}{{ else }}default value{{ end }}` (e.g. `{{ get "REPLICAS" 1 }}`).
- `--ignore-unset` CLI flag (e.g. `echo 'kind: $A$B' | kubetpl r - -s A=X --syntax=$ --ignore-unset` prints `kind: X$B`).

### Fixed
- `--freeze`ing of `initContainers[*].env[*].valueFrom.secretKeyRef.name`.

## [0.7.1](https://github.com/shyiko/kubetpl/compare/0.7.0...0.7.1) - 2018-07-16

### Fixed
- Difference in `kind: Template` rendering when `--syntax=template-kind`/`# kubetpl:syntax:template-kind` is specified (from when it's not).

## [0.7.0](https://github.com/shyiko/kubetpl/compare/0.6.0...0.7.0) - 2018-07-15

### Changed
- **BREAKING**: `kind: Template` renderer to exclude entries that evaluate to `null` ([#8](https://github.com/shyiko/kubetpl/issues/8)).

## [0.6.0](https://github.com/shyiko/kubetpl/compare/0.5.0...0.6.0) - 2018-06-02

### Added
- CRLF -> LF line separator normalization.

## [0.5.0](https://github.com/shyiko/kubetpl/compare/0.4.1...0.5.0) - 2018-04-29

### Added
- `kubetpl/data-from-env-file` extension ([#3](https://github.com/shyiko/kubetpl/issues/3)).

## [0.4.1](https://github.com/shyiko/kubetpl/compare/0.4.0...0.4.1) - 2018-04-20

### Fixed
- <kbd>Tab</kbd> completion of `args` and `-x`.

## [0.4.0](https://github.com/shyiko/kubetpl/compare/0.3.0...0.4.0) - 2018-04-20

### Fixed
- `--freeze`ing of [kubesec](https://github.com/shyiko/kubesec)-managed Secrets.

### Changed
- Empty doc rendering (empty documents are now excluded from the output regardless of `--syntax`).

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
