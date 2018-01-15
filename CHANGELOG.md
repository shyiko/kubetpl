# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/shyiko/kubetpl/compare/0.1.0...0.2.0) - 2018-01-15

### Added
- `$$` to represent literal dollar sign (`$$v` is interpreted as `$v` string and not `'$' + <value_of_v>`) (placeholder template flavor).
- line/column in error messages (placeholder template flavor).

## 0.1.0 - 2017-09-26
