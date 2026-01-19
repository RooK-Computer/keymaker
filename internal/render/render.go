package render

import (
	"context"

	"github.com/rook-computer/keymaker/internal/state"
)

type Renderer interface {
	Start(ctx context.Context) error
	Stop() error
	SetScreen(screen Screen)
	RunLoop(ctx context.Context, store *state.Store)
	RedrawWithState(snap state.State)
}

type Screen interface {
	Start(ctx context.Context) error
	Stop() error
	Draw(r Drawer, s state.State)
}

// Stub implementations
type NoopRenderer struct{}

func (n *NoopRenderer) Start(ctx context.Context) error                 { return nil }
func (n *NoopRenderer) Stop() error                                     { return nil }
func (n *NoopRenderer) SetScreen(screen Screen)                         {}
func (n *NoopRenderer) RunLoop(ctx context.Context, store *state.Store) {}
func (n *NoopRenderer) RedrawWithState(snap state.State)                {}

// Drawer is an abstraction the renderer provides to screens to draw primitives
// without exposing low-level framebuffer details.
type Drawer interface {
	FillBackground()
	DrawLogoCenteredTop()
	DrawTextCentered(text string)
}
