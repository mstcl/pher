CGO_ENABLED=0
export CGO_ENABLED

.DEFAULT_GOAL := build
.PHONY: lint tidy build

golangci-lint:
	golangci-lint run

tidy:
	go mod tidy

build:
	go build -gcflags "-l" -ldflags "-w -s" -o pher main.go

install:
	go install -gcflags "-l" -ldflags "-w -s" .
