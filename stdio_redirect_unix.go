//go:build unix

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func redirectStdIO(path string) error {
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Duplicate the file descriptor onto stdout/stderr so panics and all prints
	// (including from other goroutines) end up in the file.
	if err := unix.Dup2(int(f.Fd()), int(os.Stdout.Fd())); err != nil {
		return err
	}
	if err := unix.Dup2(int(f.Fd()), int(os.Stderr.Fd())); err != nil {
		return err
	}
	return nil
}
