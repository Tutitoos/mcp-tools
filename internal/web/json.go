package web

import (
	"bufio"
	"encoding/json"
)

// jsonEncode is a thin wrapper around encoding/json that the auth.go
// shim imports. Centralising it here keeps the auth file focused on
// middleware and lets future tests stub it out.
func jsonEncode(w *bufio.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}