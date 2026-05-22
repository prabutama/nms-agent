SHELL := /bin/sh

.PHONY: fmt test vet build check

fmt:
	go fmt ./...

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...

check: fmt test vet build
