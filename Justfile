export CGO_ENABLED := "0"

# build locally
build: tidy
	go build -gcflags "-l" -ldflags "-w -s" .

# run golangci-lint
lint:
	golangci-lint run

# run go mod tidy
tidy:
	go mod tidy

# install to user path
install: tidy build
	go install -gcflags "-l" -ldflags "-w -s" .

# run go test
test:
	go test ./...
