package screens

import (
	"context"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
)

type MainScreen struct{}

func (MainScreen) Start(ctx context.Context) error { return nil }
func (MainScreen) Stop() error                     { return nil }

func (MainScreen) Draw(r render.Drawer, st state.State) {
	r.FillBackground()
	r.DrawLogoCenteredTop()
	r.DrawTextCentered("ready")
}
