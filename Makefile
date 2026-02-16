BINARY := spanforge
PKG := ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint bench-transport docs

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/spanforge

test:
	go test $(PKG)

lint:
	go vet $(PKG)

bench-transport:
	SPANFORGE_TRANSPORT_PERF=1 go test ./internal/app -run TestTransportPerfComparison -count=1 -v

docs:
	./scripts/update-readme-flags.sh
