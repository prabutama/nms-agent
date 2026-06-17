SHELL := /bin/sh

.PHONY: fmt test vet race lint vuln build check check-all

fmt:
	go fmt ./...

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint:
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "warning: staticcheck not installed (install: go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
	fi

vuln:
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "warning: govulncheck not installed (install: go install golang.org/x/vuln/cmd/govulncheck@latest)"; \
	fi

build:
	go build ./...

check: fmt test vet build

check-all: fmt test test-race vet lint vuln build
