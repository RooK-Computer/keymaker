package render

import (
    "context"
    "github.com/rook-computer/keymaker/internal/state"
)

type Renderer interface {
    Start(ctx context.Context) error
    Stop() error
}

type Screen interface {
    Draw(s state.State)
}

// Stub implementations
type NoopRenderer struct{}

func (n *NoopRenderer) Start(ctx context.Context) error { return nil }
func (n *NoopRenderer) Stop() error { return nil }

// Screen stubs
type RemoveCartridgeScreen struct{}
func (RemoveCartridgeScreen) Draw(s state.State) {}

type InsertCartridgeScreen struct{}
func (InsertCartridgeScreen) Draw(s state.State) {}

type MainScreen struct{}
func (MainScreen) Draw(s state.State) {}
