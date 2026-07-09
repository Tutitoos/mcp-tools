//go:build unix

package cli

import (
	"bufio"
	"os"
)

// readPromptLine reads a line from /dev/tty if available, falling back to
// os.Stdin. EOF on either yields ("" , io.EOF-style error).
func readPromptLine() (string, error) {
	f := openTTY()
	if f == nil {
		f = os.Stdin
	}
	defer func() {
		if f != os.Stdin {
			f.Close()
		}
	}()
	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrInvalid
}

func openTTY() *os.File {
	for _, p := range []string{"/dev/tty"} {
		if f, err := os.Open(p); err == nil {
			return f
		}
	}
	return nil
}
