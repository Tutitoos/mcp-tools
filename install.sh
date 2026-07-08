#!/usr/bin/env bash
# One-liner installer for the mcp-tools binary.
#   curl -fsSL https://raw.githubusercontent.com/Tutitoos/mcp-tools/main/install.sh | bash
#
# Detects OS/arch, downloads the matching release tarball from GitHub, and installs
# ~/.local/bin/mcp-tools (or $MCP_TOOLS_BIN if set). Idempotent: safe to re-run.
set -euo pipefail

REPO="Tutitoos/mcp-tools"
BIN_DIR="${MCP_TOOLS_BIN:-$HOME/.local/bin}"
VERSION="${MCP_TOOLS_VERSION:-latest}"

log()  { printf '\033[36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33mwarn:\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[31merror:\033[0m %s\n' "$*" >&2; exit 1; }

command -v curl >/dev/null || err "curl no está instalado"
command -v tar  >/dev/null || err "tar no está instalado"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

# Match goreleaser's default archive naming: {os}_{arch} where arch is amd64/arm64.
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) err "arquitectura no soportada: $arch (soportadas: x86_64/amd64, aarch64/arm64)" ;;
esac
case "$os" in
  linux|darwin) ;;
  *) err "OS no soportado: $os (soportados: linux, darwin)" ;;
esac

if [ "$VERSION" = "latest" ]; then
  log "resolviendo última release en GitHub..."
  resp="$(curl -sSL -o /dev/null -w '%{http_code} %{url_effective}' "https://github.com/${REPO}/releases/latest")"
  code="${resp%% *}"
  latest="${resp#* }"
  if [ "$code" != "200" ] && [ "$code" != "301" ] && [ "$code" != "302" ]; then
    err "GitHub respondió HTTP $code al resolver 'latest' (¿rate-limit?). Reintenta en unos minutos o fija MCP_TOOLS_VERSION=vX.Y.Z"
  fi
  VERSION="${latest##*/}"
  if [ -z "$VERSION" ] || [ "$VERSION" = "releases" ]; then
    err "no pude resolver 'latest' release; ¿existe una? Si acabas de publicar, espera 1-2 min y reintenta."
  fi
fi
# Strip leading 'v' from the tag when composing the tarball filename (goreleaser default).
version_no_v="${VERSION#v}"

tarball="mcp-tools_${version_no_v}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${tarball}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

log "descargando ${tarball} (${VERSION})"
curl -fsSL "$url" -o "$tmp/pkg.tar.gz" || {
  cat >&2 <<MSG
error: no pude descargar
  $url

La release "${VERSION}" no existe (aún) en GitHub. Causas típicas:
  - GitHub Actions está deshabilitado en el repo (Settings → Actions → General → 'Allow all actions').
  - El workflow .github/workflows/release.yml falló; revisa https://github.com/${REPO}/actions.

Alternativa mientras tanto (construye desde source; requiere Go 1.24+):
  git clone https://github.com/${REPO} ~/mcp-tools
  cd ~/mcp-tools
  make install    # instala en ~/.local/bin/mcp-tools

O directamente:
  go install github.com/${REPO}/cmd/mcp-tools@latest
MSG
  exit 1
}

log "verificando checksum"
curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt" -o "$tmp/checksums.txt" \
  || err "no pude descargar checksums.txt de la release ${VERSION}"
expected="$(grep " ${tarball}\$" "$tmp/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || err "checksums.txt no contiene ${tarball}"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/pkg.tar.gz" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/pkg.tar.gz" | awk '{print $1}')"
else
  err "necesita sha256sum o shasum en PATH para verificar checksum"
fi
[ "$actual" = "$expected" ] || err "checksum mismatch: esperado $expected, actual $actual"

log "extrayendo binario"
tar -xzf "$tmp/pkg.tar.gz" -C "$tmp" mcp-tools \
  || err "no encontré 'mcp-tools' dentro del tarball"

mkdir -p "$BIN_DIR"
install -m 0755 "$tmp/mcp-tools" "$BIN_DIR/mcp-tools"
log "instalado $BIN_DIR/mcp-tools"

# PATH sanity check
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    warn "$BIN_DIR no está en tu \$PATH."
    warn "Añade a tu shell rc (~/.bashrc o ~/.zshrc):"
    # literal instruction text for the user to paste, not expanded here
    # shellcheck disable=SC2016
    printf '\n    export PATH="%s:$PATH"\n\n' "$BIN_DIR" >&2
    ;;
esac

"$BIN_DIR/mcp-tools" --version
cat <<MSG

Siguiente paso:
  git clone https://github.com/${REPO} \${HOME}/mcp-tools
  cd \${HOME}/mcp-tools
  $BIN_DIR/mcp-tools install
MSG
