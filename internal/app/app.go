package app

import (
    "context"
    "github.com/rook-computer/keymaker/internal/buttons"
    "github.com/rook-computer/keymaker/internal/flash"
    "github.com/rook-computer/keymaker/internal/render"
    "github.com/rook-computer/keymaker/internal/state"
    "github.com/rook-computer/keymaker/internal/web"
)

type App struct {
    Store   *state.Store
    Render  render.Renderer
    Web     web.Server
    Flash   flash.Flasher
    Buttons buttons.Buttons
}

func New(store *state.Store, r render.Renderer, w web.Server, f flash.Flasher, b buttons.Buttons) *App {
    return &App{Store: store, Render: r, Web: w, Flash: f, Buttons: b}
}

func (a *App) Start(ctx context.Context) error {
    a.Store.SetPhase(state.READY)
    // Start subsystems in the future; for now no-op
    return nil
}

func (a *App) Stop() error {
    // Stop subsystems in the future; for now no-op
    return nil
}
