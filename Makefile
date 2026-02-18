BINARY ?= ytcli
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/CoastalFuturist/ytcli/internal/buildinfo.Version=$(VERSION) -X github.com/CoastalFuturist/ytcli/internal/buildinfo.Commit=$(COMMIT) -X github.com/CoastalFuturist/ytcli/internal/buildinfo.Date=$(DATE)

.PHONY: build test fmt clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/ytcli

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal ./main.go

clean:
	rm -rf $(BINARY) dist
