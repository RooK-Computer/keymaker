package screens

import (
	"context"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
)

type InsertCartridgeScreen struct{}

func (InsertCartridgeScreen) Start(ctx context.Context) error { return nil }
func (InsertCartridgeScreen) Stop() error                     { return nil }

func (InsertCartridgeScreen) Draw(r render.Drawer, st state.State) {
	r.FillBackground()
	r.DrawLogoCenteredTop()
	r.DrawTextCentered("please insert cartridge")
}
