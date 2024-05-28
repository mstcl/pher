.DEFAULT_GOAL := build
.PHONY: lint tidy build

golangci-lint:
	golangci-lint run

tidy:
	go mod tidy

build:
	go build -ldflags "-w -s" -o pher main.go

install:
	go install -ldflags "-w -s" main.go
