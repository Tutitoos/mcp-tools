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

PNPM := $(shell command -v pnpm 2>/dev/null)
ifeq ($(PNPM),)
PNPM := $(HOME)/.nvm/versions/node/$(shell node -v 2>/dev/null | sed 's/^v//' | cut -d. -f1 | xargs -I{} echo {} || echo "")/bin/pnpm
endif

# Copy (not symlink) web/build into the webassets package directory so the
# //go:embed directive (which cannot follow symlinks) can see the built
# SPA bundle. Idempotent: removes the prior copy first. Cheap: the bundle
# is ~1 MB and `cp -rL` dereferences symlinks so the copy is plain files.
webassets/build:
	@rm -rf webassets/build
	@mkdir -p webassets/build
	@cp -rL web/build/. webassets/build/
	@echo "webassets: copied web/build -> webassets/build"
.PHONY: build build-web dev dev-web install test release clean web-bootstrap

build: webassets/build web-bootstrap
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/mcp-tools

build-web:
	cd web && $(PNPM) install --frozen-lockfile=false && $(PNPM) run build

# Ensures `web/build/client/` exists with at least one file so the
# `//go:embed all:web/build/client` directive in webassets.go compiles even
# when the SPA hasn't been built (e.g. CI Go-only jobs).
web-bootstrap:
	@mkdir -p web/build/client
	@if [ ! -f web/build/client/.keep ]; then \
		echo '<!doctype html><html><body>mcp-tools web admin panel (build web/ first)</body></html>' > web/build/client/index.html; \
		touch web/build/client/.keep; \
	fi

dev-web:
	cd web && $(PNPM) run dev

dev: web-bootstrap
	$(GO) run ./cmd/mcp-tools serve --port 8080 & \
	cd web && $(PNPM) run dev

install: build
	install -m 0755 bin/$(BINARY) $${MCP_TOOLS_BIN:-$$HOME/.local/bin}/$(BINARY)

test:
	$(GO) test ./...

release:
	goreleaser release --clean

clean:
	rm -rf bin/
	rm -rf web/build
	rm -rf web/node_modules