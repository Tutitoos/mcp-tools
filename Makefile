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
# nvm keeps the `v` prefix in its directory names (~/.nvm/versions/node/v22.14.0/).
PNPM := $(HOME)/.nvm/versions/node/$(shell node -v 2>/dev/null)/bin/pnpm
endif

# Copy (not symlink) web/build into the webassets package directory so the
webassets/build: web/build/client/index.html
	@rm -rf $@
	@mkdir -p $@
	@cp -rL web/build/. $@/
	@echo "webassets: copied web/build -> $@"

# Real SPA build (client + server bundles). Touches `.keep` so
# `web-bootstrap` knows to skip. The Vite config splits by `isSsrBuild`
# so the same `pnpm run build` invocation produces both bundles when
# invoked with `--ssr`; we run it twice (once for the client, once
# for the server) because a single `vite build` cannot produce two
# outputs.
web/build/client/index.html web/build/server/index.js:
	cd web && $(PNPM) install --frozen-lockfile=false && \
	  $(PNPM) run build && \
	  $(PNPM) exec vite build --ssr
	@touch web/build/client/.keep
.PHONY: build-web
build-web: web/build/client/index.html web/build/server/index.js
	@echo "build-web: client + SSR bundles ready"

# Placeholder for Go-only CI jobs (no SPA). Creates the bare minimum
# under web/build/client/ so `//go:embed all:build/client` compiles.
# Touches `.keep` so subsequent `make build` invocations skip this.
.PHONY: web-bootstrap
web-bootstrap:
	@mkdir -p web/build/client
	@if [ ! -f web/build/client/.keep ]; then \
		echo '<!doctype html><html><body>mcp-tools web admin panel (build web/ first)</body></html>' > web/build/client/index.html; \
		touch web/build/client/.keep; \
	fi

.PHONY: build dev dev-web install test release clean

build: webassets/build
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/mcp-tools

dev-web:
	cd web && $(PNPM) run dev

dev: web-bootstrap
	$(GO) run ./cmd/mcp-tools serve --port 8080 & \
	cd web && $(PNPM) run dev

install: build
	install -m 0755 bin/$(BINARY) $${MCP_TOOLS_BIN:-$$HOME/.local/bin}/$(BINARY)
	@$${MCP_TOOLS_BIN:-$$HOME/.local/bin}/$(BINARY) web --restart || true

test:
	$(GO) test ./...

release:
	goreleaser release --clean

clean:
	rm -rf bin/
	rm -rf web/build
	rm -rf web/node_modules