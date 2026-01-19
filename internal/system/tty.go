package system

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// KD console modes from linux/kd.h
const (
	kdText     = 0x00
	kdGraphics = 0x01
	kdSetMode  = 0x4B3A // KDSETMODE ioctl
)

// SetGraphicsMode switches the active console to graphics mode to suppress the hardware cursor.
// It targets /dev/tty0, the current active virtual terminal.
func SetGraphicsMode() error {
	// Prefer /dev/tty (active VT), fallback to /dev/tty0
	paths := []string{"/dev/tty", "/dev/tty0"}
	var lastErr error
	for _, path := range paths {
		fd, err := unix.Open(path, unix.O_RDONLY, 0)
		if err != nil {
			lastErr = fmt.Errorf("open %s: %w", path, err)
			continue
		}
		defer unix.Close(fd)
		if err := unix.IoctlSetInt(fd, kdSetMode, kdGraphics); err != nil {
			lastErr = fmt.Errorf("KD_GRAPHICS on %s: %w", path, err)
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("KD_GRAPHICS failed: unknown error")
}

// Logging wrappers
type logger interface {
	Infof(string, string, ...interface{})
	Errorf(string, string, ...interface{})
}

func SetGraphicsModeWithLog(logger logger) error {
	err := SetGraphicsMode()
	if err != nil {
		if logger != nil {
			logger.Errorf("tty", "KD_GRAPHICS failed: %v", err)
		}
	} else {
		if logger != nil {
			logger.Infof("tty", "KD_GRAPHICS set")
		}
	}
	return err
}

// RestoreTextMode restores the console to text mode so cursor and normal console return.
func RestoreTextMode() error {
	paths := []string{"/dev/tty", "/dev/tty0"}
	var lastErr error
	for _, path := range paths {
		fd, err := unix.Open(path, unix.O_RDONLY, 0)
		if err != nil {
			lastErr = fmt.Errorf("open %s: %w", path, err)
			continue
		}
		defer unix.Close(fd)
		if err := unix.IoctlSetInt(fd, kdSetMode, kdText); err != nil {
			lastErr = fmt.Errorf("KD_TEXT on %s: %w", path, err)
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("KD_TEXT failed: unknown error")
}

func RestoreTextModeWithLog(logger logger) error {
	err := RestoreTextMode()
	if err != nil {
		if logger != nil {
			logger.Errorf("tty", "KD_TEXT failed: %v", err)
		}
	} else {
		if logger != nil {
			logger.Infof("tty", "KD_TEXT set")
		}
	}
	return err
}

// HideCursor writes the ANSI escape to hide the cursor to the active VT.
func HideCursor() error { return writeVT("\x1b[?25l") }
func ShowCursor() error { return writeVT("\x1b[?25h") }
func HideCursorWithLog(logger logger) error {
	err := HideCursor()
	if err != nil {
		if logger != nil {
			logger.Errorf("tty", "hide cursor failed: %v", err)
		}
	} else {
		if logger != nil {
			logger.Infof("tty", "cursor hidden")
		}
	}
	return err
}
func ShowCursorWithLog(logger logger) error {
	err := ShowCursor()
	if err != nil {
		if logger != nil {
			logger.Errorf("tty", "show cursor failed: %v", err)
		}
	} else {
		if logger != nil {
			logger.Infof("tty", "cursor shown")
		}
	}
	return err
}

func writeVT(s string) error {
	// Try /dev/tty first
	paths := []string{"/dev/tty", "/dev/tty0"}
	var lastErr error
	for _, path := range paths {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			lastErr = err
			continue
		}
		defer f.Close()
		_, err = f.WriteString(s)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return fmt.Errorf("write VT failed: %v", lastErr)
	}
	return fmt.Errorf("write VT failed: unknown error")
}
