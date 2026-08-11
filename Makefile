.PHONY: build build-linux test verify

build:
	go build -o bin/psl-test-logs-mcp ./cmd/psl-test-logs-mcp

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/psl-test-logs-mcp-linux-amd64 ./cmd/psl-test-logs-mcp

test:
	go test ./...

verify:
	go test -race ./...
	go vet ./...
	go build -o bin/psl-test-logs-mcp ./cmd/psl-test-logs-mcp
