package screens

import (
	"context"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
)

type MainScreen struct{}

func (MainScreen) Start(ctx context.Context) error { return nil }
func (MainScreen) Stop() error                     { return nil }

func (MainScreen) Draw(drawer render.Drawer, currentState state.State) {
	drawer.FillBackground()
	drawer.DrawLogoCenteredTop()
	drawer.DrawText("status", 40, 40, render.TextStyle{Size: 24, Align: render.TextAlignLeft})
	drawer.DrawTextCentered("ready")
}
