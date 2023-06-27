# goboundcheck

**_Go linter that validates all accesses to slices and arrays are bound-checked._**

[![Go](https://github.com/morgenm/goboundcheck/actions/workflows/go.yml/badge.svg)](https://github.com/morgenm/goboundcheck/actions/workflows/go.yml)
[![golangci-lint](https://github.com/morgenm/goboundcheck/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/morgenm/goboundcheck/actions/workflows/golangci-lint.yml)
[![codecov](https://codecov.io/gh/morgenm/goboundcheck/branch/main/graph/badge.svg?token=5CGBX5Q5NC)](https://codecov.io/gh/morgenm/goboundcheck)

## About
Go linter which warns of any slice or array accesses which are not enclosed in an if-statement that validates capacity or length. These warnings are meant to help notify developers which accesses aren't bound-checked to help prevent out-of-bound runtime errors. 

The idea for this comes from rule G602 which I contributed to [gosec](https://github.com/securego/gosec). That rule only validates slices whose capacities are determined by calls to `make()` where the capacity/length is a constant literal, or by reslicing slices made with `make()`. This linter is simpler than that rule, as it flags all slice and array accesses which are made without first checking capacity or length. I made *goboundcheck* for developers who want to have strict validation on all slices and arrays. If you want a less noisey and less strict bound-checker, check out *gosec*.

## Install

### Building Locally 
```bash
git clone https://github.com/morgenm/goboundcheck
make
```
This will output the executable file `goboundcheck` on Linux or Mac, and `goboundcheck.exe` on Windows

## Usage
The syntax of `goboundcheck` is similar to other Go linters, due to being built off of `golang.org/x/tools/go/analysis`. 

To recursively scan code starting in the current directory:
```bash
goboundcheck ./...
``` 

You can also scan specific files:
```bash
goboundcheck code1.go code2.ggo
```

To see other flags and options:
```bash
goboundcheck --help
```