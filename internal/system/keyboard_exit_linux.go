//go:build linux

package system

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	evKey = 0x01

	// Linux input-event-codes.h
	keyF4 = 62
)

type keyboardExitLogger interface {
	Infof(string, string, ...interface{})
	Errorf(string, string, ...interface{})
}

// StartExitOnF4 watches Linux evdev devices under /dev/input/event* and invokes onExit
// once when the F4 key is pressed.
//
// It is best-effort: if no input devices are available, it logs and returns.
func StartExitOnF4(ctx context.Context, logger keyboardExitLogger, onExit func()) {
	if onExit == nil {
		return
	}

	// Determine input_event size based on arch timeval size.
	// input_event = timeval + u16 type + u16 code + s32 value.
	tvSize := int(binary.Size(unix.Timeval{}))
	eventSize := tvSize + 2 + 2 + 4
	if eventSize <= 0 {
		eventSize = 24
	}

	paths, err := filepath.Glob("/dev/input/event*")
	if err != nil || len(paths) == 0 {
		if logger != nil {
			logger.Infof("input", "no evdev devices found for F4 exit")
		}
		return
	}

	var once sync.Once
	triggerExit := func() {
		once.Do(func() {
			if logger != nil {
				logger.Infof("input", "F4 pressed: exiting")
			}
			onExit()
		})
	}

	for _, path := range paths {
		p := path
		go func() {
			// Resiliency: evdev devices can be transient or return errors.
			// Keep retrying until the app context is canceled.
			buf := make([]byte, 4096)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				fd, err := unix.Open(p, unix.O_RDONLY|unix.O_NONBLOCK, 0)
				if err != nil {
					// Device may not exist yet or permissions may fluctuate.
					time.Sleep(500 * time.Millisecond)
					continue
				}
				f := os.NewFile(uintptr(fd), p)

				err = func() error {
					defer func() { _ = f.Close() }()
					for {
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
						}

						pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
						_, pollErr := unix.Poll(pollFds, 250)
						if pollErr != nil {
							return pollErr
						}
						if pollFds[0].Revents&unix.POLLIN == 0 {
							continue
						}

						n, readErr := unix.Read(fd, buf)
						if readErr != nil {
							if readErr == unix.EAGAIN || readErr == unix.EINTR {
								continue
							}
							return readErr
						}
						if n < eventSize {
							continue
						}

						// Parse as a sequence of input_event records.
						for off := 0; off+eventSize <= n; off += eventSize {
							rec := buf[off : off+eventSize]
							// type and code are immediately after timeval.
							typ := binary.LittleEndian.Uint16(rec[tvSize : tvSize+2])
							code := binary.LittleEndian.Uint16(rec[tvSize+2 : tvSize+4])
							value := int32(binary.LittleEndian.Uint32(rec[tvSize+4 : tvSize+8]))
							if typ == evKey && code == keyF4 && value == 1 {
								triggerExit()
								// Give the app a moment to unwind; then stop reading.
								time.Sleep(50 * time.Millisecond)
								return nil
							}
						}
					}
				}()

				if err == nil {
					// Exit was requested.
					return
				}
				if ctx.Err() != nil {
					return
				}
				if logger != nil {
					logger.Errorf("input", "evdev F4 watcher stopped for %s: %v (retrying)", p, err)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}()
	}
}

// StartExitOnF4TTY is a fallback watcher that reads keypresses from the active TTY.
// This can be more reliable than evdev in some environments.
//
// It switches /dev/tty into a raw-ish mode (no canonical buffering) for the duration
// of the watcher and restores settings on exit.
func StartExitOnF4TTY(ctx context.Context, logger keyboardExitLogger, onExit func()) {
	if onExit == nil {
		return
	}

	var once sync.Once
	triggerExit := func() {
		once.Do(func() {
			if logger != nil {
				logger.Infof("input", "F4 pressed (tty): exiting")
			}
			onExit()
		})
	}

	go func() {
		// Resiliency: the underlying TTY can change or temporarily error.
		// Re-open and re-apply raw mode on failure.
		buf := make([]byte, 64)
		var window []byte

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			f, shouldClose := openTTYForRead(logger)
			if f == nil {
				time.Sleep(750 * time.Millisecond)
				continue
			}
			fd := int(f.Fd())
			oldState, ok := makeRaw(fd)
			if !ok {
				if shouldClose {
					_ = f.Close()
				}
				if logger != nil {
					logger.Errorf("input", "tty F4 watcher: termios unavailable (retrying)")
				}
				time.Sleep(750 * time.Millisecond)
				continue
			}

			window = window[:0]
			err := func() error {
				defer func() {
					_ = restoreTermios(fd, oldState)
					if shouldClose {
						_ = f.Close()
					}
				}()

				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}

					pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
					_, pollErr := unix.Poll(pollFds, 250)
					if pollErr != nil {
						return pollErr
					}
					if pollFds[0].Revents&unix.POLLIN == 0 {
						continue
					}

					n, readErr := unix.Read(fd, buf)
					if readErr != nil {
						if readErr == unix.EAGAIN || readErr == unix.EINTR {
							continue
						}
						return readErr
					}
					if n <= 0 {
						continue
					}

					window = append(window, buf[:n]...)
					if len(window) > 32 {
						window = window[len(window)-32:]
					}

					if containsF4Sequence(window) {
						triggerExit()
						return nil
					}
				}
			}()

			if err == nil {
				return
			}
			if ctx.Err() != nil {
				return
			}
			if logger != nil {
				logger.Errorf("input", "tty F4 watcher stopped: %v (retrying)", err)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

func openTTYForRead(logger keyboardExitLogger) (file *os.File, shouldClose bool) {
	// Prefer stdin if it is a real TTY. This matches the user's expectation when
	// running the app in the foreground.
	if stdin := os.Stdin; stdin != nil {
		if _, err := unix.IoctlGetTermios(int(stdin.Fd()), unix.TCGETS); err == nil {
			if logger != nil {
				logger.Infof("input", "tty F4 watcher using stdin")
			}
			return stdin, false
		}
	}

	// When started as a service, stdin is often not connected to a terminal.
	// Fall back to the controlling terminal /dev/tty and then VT /dev/tty0.
	for _, path := range []string{"/dev/tty", "/dev/tty0"} {
		f, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			continue
		}
		if logger != nil {
			logger.Infof("input", "tty F4 watcher using %s", path)
		}
		return f, true
	}

	if logger != nil {
		logger.Infof("input", "tty F4 watcher disabled (no usable TTY on stdin or /dev/tty*)")
	}
	return nil, false
}

func makeRaw(fd int) (*unix.Termios, bool) {
	oldState, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, false
	}

	newState := *oldState

	// Rough cfmakeraw(). Keep it minimal.
	newState.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	newState.Oflag &^= unix.OPOST
	newState.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	newState.Cflag &^= unix.CSIZE | unix.PARENB
	newState.Cflag |= unix.CS8
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newState); err != nil {
		return nil, false
	}
	return oldState, true
}

func restoreTermios(fd int, oldState *unix.Termios) error {
	if oldState == nil {
		return nil
	}
	return unix.IoctlSetTermios(fd, unix.TCSETS, oldState)
}

func containsF4Sequence(b []byte) bool {
	// Common terminals:
	// - xterm: ESC O S
	// - vt/linux console: ESC [ 1 4 ~
	for i := 0; i+2 < len(b); i++ {
		if b[i] == 0x1b && b[i+1] == 'O' && b[i+2] == 'S' {
			return true
		}
	}

	for i := 0; i < len(b); i++ {
		if b[i] != 0x1b {
			continue
		}
		if i+2 >= len(b) || b[i+1] != '[' {
			continue
		}
		j := i + 2
		for j < len(b) && (b[j] < '0' || b[j] > '9') {
			j++
		}
		start := j
		for j < len(b) && b[j] >= '0' && b[j] <= '9' {
			j++
		}
		if start == j || j >= len(b) {
			continue
		}
		// Typical form ESC [ 14 ~
		if b[j] == '~' {
			code, err := strconv.Atoi(string(b[start:j]))
			if err == nil && code == 14 {
				return true
			}
		}
	}

	return false
}
