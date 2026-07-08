BINARY := mcp-tools
LDFLAGS := -X github.com/Tutitoos/mcp-tools/internal/version.Version=$(shell git describe --tags --always 2>/dev/null || echo dev) \
           -X github.com/Tutitoos/mcp-tools/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown) \
           -X github.com/Tutitoos/mcp-tools/internal/version.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Resolve `go`: prefer PATH, fall back to the install.sh bootstrap location
# ($HOME/.local/go/bin) so `make install` works without re-sourcing the rc.
GO := $(shell command -v go 2>/dev/null)
ifeq ($(GO),)
GO := $(HOME)/.local/go/bin/go
endif

.PHONY: build install test release clean

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/mcp-tools

install: build
	install -m 0755 bin/$(BINARY) $${MCP_TOOLS_BIN:-$$HOME/.local/bin}/$(BINARY)

test:
	$(GO) test ./...

release:
	goreleaser release --clean

clean:
	rm -rf bin/
