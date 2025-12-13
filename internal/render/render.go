package render

import (
    "context"
    "github.com/rook-computer/keymaker/internal/state"
)

type Renderer interface {
    Start(ctx context.Context) error
    Stop() error
    SetScreen(screen Screen)
    Redraw()
}

type Screen interface {
    Draw(r Drawer, s state.State)
}

// Stub implementations
type NoopRenderer struct{}

func (n *NoopRenderer) Start(ctx context.Context) error { return nil }
func (n *NoopRenderer) Stop() error { return nil }
func (n *NoopRenderer) SetScreen(screen Screen) {}
func (n *NoopRenderer) Redraw() {}

// Screen stubs
type RemoveCartridgeScreen struct{}
func (RemoveCartridgeScreen) Draw(r Drawer, s state.State) {
    // Fill background
    r.FillBackground()
    // Draw logo and message
    r.DrawLogoCenteredTop()
    r.DrawTextCentered("please remove cartridge")
}

type InsertCartridgeScreen struct{}
func (InsertCartridgeScreen) Draw(r Drawer, s state.State) {}

type MainScreen struct{}
func (MainScreen) Draw(r Drawer, s state.State) {}

// Drawer is an abstraction the renderer provides to screens to draw primitives
// without exposing low-level framebuffer details.
type Drawer interface {
    FillBackground()
    DrawLogoCenteredTop()
    DrawTextCentered(text string)
}
