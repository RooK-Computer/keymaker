//go:build !unix

package main

import "os"

// Best-effort fallback for non-Unix platforms.
// Note: this does not reliably capture runtime-level stderr output (like panics)
// the same way Dup2 does on Unix, but it keeps builds working.
func redirectStdIO(path string) error {
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	os.Stdout = f
	os.Stderr = f
	return nil
}
