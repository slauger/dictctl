PREFIX ?= /usr/local
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: build install clean lint

build:
	go build -ldflags "$(LDFLAGS)" -o dictctl ./cmd/dictctl

install: build
	install -m 755 dictctl $(PREFIX)/bin/dictctl

lint:
	golangci-lint run
	govulncheck ./...

clean:
	rm -f dictctl
	rm -rf dist/
