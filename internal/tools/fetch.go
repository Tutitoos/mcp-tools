package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetchVerified downloads url and verifies its SHA-256 against wantHex
// before returning the bytes. It is the guard for the `install.sh | sh`
// third-party installer (codebase-memory): its URL is pinned
// to a commit SHA, and this checksum additionally protects against a
// compromised CDN or a rewritten object. On mismatch nothing is executed.
//
// Bumping a pinned installer = update the commit in the URL *and* the
// checksum constant next to it, after reviewing the new script.
func fetchVerified(url, wantHex string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("descargar %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("descargar %s: HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // installers are a few KB; 4 MiB is a hard sanity cap
	if err != nil {
		return nil, fmt.Errorf("leer %s: %w", url, err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != wantHex {
		return nil, fmt.Errorf("checksum de %s no coincide: esperado %s, obtenido %s — upstream cambió el script; revísalo y actualiza el pin", url, wantHex, got)
	}
	return data, nil
}
