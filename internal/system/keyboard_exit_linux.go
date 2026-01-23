//go:build linux

package system

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
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
			fd, err := unix.Open(p, unix.O_RDONLY|unix.O_NONBLOCK, 0)
			if err != nil {
				return
			}
			f := os.NewFile(uintptr(fd), p)
			defer func() {
				_ = f.Close()
			}()

			buf := make([]byte, 4096)

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
				_, pollErr := unix.Poll(pollFds, 250)
				if pollErr != nil {
					// Device might have gone away.
					return
				}
				if pollFds[0].Revents&unix.POLLIN == 0 {
					continue
				}

				n, readErr := unix.Read(fd, buf)
				if readErr != nil {
					if readErr == unix.EAGAIN || readErr == unix.EINTR {
						continue
					}
					return
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
						return
					}
				}
			}
		}()
	}
}
