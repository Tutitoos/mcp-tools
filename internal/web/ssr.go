package web

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// shellTemplate is the document shell used to wrap SSR output. It is
// loaded once at startup from build/client/index.html (Vite's built
// template) and contains the canonical <html>/<head> structure plus
// the script + CSS link tags. SSR produces just the <body> content
// (Shell + matched route); injectBody splices it into the template so
// the client hydrates only the body, avoiding head-content mismatches.
var shellTemplate string

// ssrEngine keeps a long-lived Node sidecar (one process per Go
// binary) that listens on 127.0.0.1:<random> and answers POST
// /render. This replaces the per-request `node <bundle> <url>` exec
// model that paid a full cold-start + ESM module load on every
// request (≈8.5s on the dashboard). The sidecar boots once at
// startup; render() reuses a keep-alive HTTP client and falls
// through to the SPA shell on any error or timeout.
const (
	sidecarHandshakeTimeout = 3 * time.Second
	sidecarRequestTimeout   = 2 * time.Second
)
type ssrEngine struct {
	cmd        *exec.Cmd
	sidecarURL string
	httpClient *http.Client
}

// renderTimeout is a per-request ceiling for the sidecar call.
// A slow SSR is worse than no SSR because the user sees a 2s blank
// page otherwise — falling through to the SPA shell is intentional.
var renderTimeout = sidecarRequestTimeout

// bundleFS is the read-only filesystem the SSR bundle lives in. We use
// an interface so tests can substitute an in-memory FS without touching
// disk.
type bundleFS interface {
	ReadFile(name string) ([]byte, error)
}

func newSSREngine(bundle bundleFS, root string) (*ssrEngine, error) {
	data, err := bundle.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("ssr: read embed: %w", err)
	}
	dir, err := os.MkdirTemp("", "mcp-tools-ssr-*")
	if err != nil {
		return nil, fmt.Errorf("ssr: tempdir: %w", err)
	}
	p := filepath.Join(dir, "index.mjs")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("ssr: write bundle: %w", err)
	}
	cmd := exec.Command("node", p, "--serve")
	// The SSR bundle has all dependencies inlined via Vite's
	// `ssr.noExternal: true`, so no node_modules tree is required at
	// runtime. NODE_NO_WARNINGS keeps stdout clean of Node's own
	// deprecation notices.
	cmd.Env = append(os.Environ(), "NODE_NO_WARNINGS=1")
	// Start the sidecar in the background and parse the READY
	// handshake to learn the listening port. Stderr stays on the
	// child's fd (inherited) so any crash shows up in the same
	// stream as the parent's logs.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("ssr: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("ssr: spawn node: %w", err)
	}
	sidecarURL, handshakeErr := readSidecarHandshake(stdout)
	if handshakeErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("ssr: handshake: %w", handshakeErr)
	}
	// Reuse a single keep-alive connection. The Transport's idle
	// connection pool is unbounded by default; one long-lived sidecar
	// keeps at most one socket warm.
	tr := &http.Transport{
		IdleConnTimeout: 60 * time.Second,
	}
	return &ssrEngine{
		cmd:        cmd,
		sidecarURL: sidecarURL,
		httpClient: &http.Client{Transport: tr, Timeout: renderTimeout},
	}, nil
}

// readSidecarHandshake parses the single "READY 127.0.0.1:<port>"
// line printed by entry.server.tsx on sidecar startup. The line is
// produced by process.stdout.write and ends with a newline.
func readSidecarHandshake(r io.Reader) (string, error) {
	type result struct {
		url string
		err error
	}
	done := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			const prefix = "READY "
			if strings.HasPrefix(line, prefix) {
				done <- result{url: "http://" + strings.TrimPrefix(line, prefix) + "/render"}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			done <- result{err: fmt.Errorf("read: %w", err)}
			return
		}
		done <- result{err: fmt.Errorf("sidecar exited before READY handshake")}
	}()
	select {
	case r := <-done:
		return r.url, r.err
	case <-time.After(sidecarHandshakeTimeout):
		return "", fmt.Errorf("timed out after %s", sidecarHandshakeTimeout)
	}
}

func (e *ssrEngine) render(urlPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.sidecarURL, strings.NewReader(urlPath))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "text/plain; charset=utf-8")
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sidecar status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

func (e *ssrEngine) Close() error {
	// Kill the node sidecar and reap it so the OS releases the
	// bundle file (the running process holds an open fd). The
	// temp dir cleanup is best-effort: a leak here only persists
	// until the Go process exits, at which point the OS sweeps
	// /tmp.
	if e != nil && e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
		_ = os.RemoveAll(filepath.Dir(e.cmd.Path))
	}
	return nil
}

// ensureNode checks that `node` is in PATH. Called once at startup.
func ensureNode() error {
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("node binary not found in PATH; required for SSR. Install Node.js >= 20 or run the web binary without the SSR bundle (SPA-only)")
	}
	return nil
}

// ensureSSRBundleExists checks the SSR bundle exists in the embed. We
// do NOT panic when it's missing: the web binary should still boot in
// SPA-only mode for hosts where the JS bundle wasn't built. Callers
// detect the nil engine and fall back to the SPA handler.
func ensureSSRBundleExists(bundle bundleFS, root string) error {
	if _, err := bundle.ReadFile(root); err != nil {
		return fmt.Errorf("ssr: bundle %s not present in embed: %w", root, err)
	}
	return nil
}

// loadShellTemplate reads the document template from the embed. Called
// once at startup; the result is cached in shellTemplate for fast
// per-request injection.
func loadShellTemplate(bundle bundleFS, root string) error {
	data, err := bundle.ReadFile(root)
	if err != nil {
		return fmt.Errorf("ssr: shell template: %w", err)
	}
	shellTemplate = string(data)
	return nil
}

// injectBody replaces the placeholder marker inside the document
// template with the SSR-rendered body. The marker is `<!--SSR-->`,
// which is added to web/index.html during the build. Falls back to
// appending the body inside <body> when the marker is absent, so the
// behaviour degrades gracefully on older builds.
func injectBody(template, body string) string {
	const marker = "<!--SSR-->"
	if strings.Contains(template, marker) {
		return strings.Replace(template, marker, body, 1)
	}
	// Fallback: inject right before </body>.
	if idx := strings.LastIndex(template, "</body>"); idx >= 0 {
		return template[:idx] + body + template[idx:]
	}
	return template + body
}