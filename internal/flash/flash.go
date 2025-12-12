package flash

import (
    "context"
    "io"
    "github.com/rook-computer/keymaker/internal/state"
)

type Flasher interface {
    Start(ctx context.Context, r io.Reader) error
    Cancel() error
    Status() state.FlashInfo
}

type NoopFlasher struct{}

func (NoopFlasher) Start(ctx context.Context, r io.Reader) error { return nil }
func (NoopFlasher) Cancel() error { return nil }
func (NoopFlasher) Status() state.FlashInfo { return state.FlashInfo{} }
