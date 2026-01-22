package flash

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/rook-computer/keymaker/internal/state"
)

// ScriptFlasher streams input into `sudo flash.sh`.
// The script is expected to read a gzipped disk image from stdin and write it to the cartridge device.
type ScriptFlasher struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	status state.FlashInfo
}

func NewScriptFlasher() *ScriptFlasher {
	return &ScriptFlasher{status: state.FlashInfo{Status: "idle"}}
}

func (f *ScriptFlasher) Start(ctx context.Context, reader io.Reader) error {
	f.mu.Lock()
	if f.cmd != nil {
		f.mu.Unlock()
		return fmt.Errorf("flash already running")
	}
	f.status = state.FlashInfo{Status: "starting"}
	cmd := exec.CommandContext(ctx, "sudo", "flash.sh")
	cmd.Stdin = reader
	cmd.Stdout = io.Discard
	stderr := &ringBuffer{max: 4096}
	cmd.Stderr = stderr
	f.cmd = cmd
	f.mu.Unlock()

	if err := cmd.Start(); err != nil {
		f.mu.Lock()
		f.cmd = nil
		f.status = state.FlashInfo{Status: "error", Err: err.Error()}
		f.mu.Unlock()
		return err
	}

	f.mu.Lock()
	f.status = state.FlashInfo{Status: "running"}
	f.mu.Unlock()

	err := cmd.Wait()

	f.mu.Lock()
	f.cmd = nil
	if err != nil {
		msg := err.Error()
		if s := stderr.String(); s != "" {
			msg = msg + ": " + s
		}
		f.status = state.FlashInfo{Status: "error", Err: msg}
		f.mu.Unlock()
		return fmt.Errorf("flash failed: %s", msg)
	}
	f.status = state.FlashInfo{Status: "done"}
	f.mu.Unlock()
	return nil
}

func (f *ScriptFlasher) Cancel() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cmd == nil || f.cmd.Process == nil {
		return nil
	}
	return f.cmd.Process.Kill()
}

func (f *ScriptFlasher) Status() state.FlashInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

type ringBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.max <= 0 {
		return len(p), nil
	}

	if len(p) >= r.max {
		r.buf = append(r.buf[:0], p[len(p)-r.max:]...)
		return len(p), nil
	}

	if len(r.buf)+len(p) > r.max {
		drop := len(r.buf) + len(p) - r.max
		r.buf = append(r.buf[drop:], p...)
		return len(p), nil
	}

	r.buf = append(r.buf, p...)
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return string(r.buf)
}
