BINARY ?= ima-mcp-server
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test vet fmt clean

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ima-mcp-server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

clean:
	rm -rf bin dist
