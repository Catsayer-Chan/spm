version    := env_var_or_default("version", "")
versionArg := if version != "" { " -X main.VERSION={{version}}" } else { "" }
program    := "spm"

alias b := build
alias c := clean
alias d := debug

# 默认执行编译
default: build

# 编译项目
build: lint
    @echo
    @echo "Building project ..."
    go build -ldflags="-w -s -extldflags '-static'{{versionArg}}" -o ./bin/{{program}} .
    @chmod +x ./bin/{{program}}
    @echo "Build success."

# 保留调试信息
debug: lint
    @echo
    go build -ldflags="-extldflags" -o ./bin/{{program}} .
    @chmod +x ./bin/{{program}}
    @echo "Build success."

lint: deps
    #!/usr/bin/env bash
    set -euxo pipefail
    go vet ./...
    if type errcheck 2>/dev/null; then
        errcheck ./...
    fi
    if type staticcheck 2>/dev/null; then
        staticcheck -checks inherit,-U1000 ./...
    fi

deps:
    go fmt ./...
    go mod tidy
    @echo

test:
    go test

clean:
    rm -f bin/spm
