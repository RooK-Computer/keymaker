package app

import (
    "context"
    "errors"
    "time"
    "github.com/rook-computer/keymaker/internal/buttons"
    "github.com/rook-computer/keymaker/internal/flash"
    "github.com/rook-computer/keymaker/internal/render"
    "github.com/rook-computer/keymaker/internal/state"
    "github.com/rook-computer/keymaker/internal/web"
    "github.com/rook-computer/keymaker/internal/system"
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
    // Initialize renderer and draw first screen
    fb := render.NewFBRenderer()
    if err := fb.Start(ctx); err != nil { return err }
    defer fb.Stop()
    fb.SetScreen(render.RemoveCartridgeScreen{})
    fb.Redraw()

    // Begin ejection process and wait with timeout retries
    // Use ShellRunner so commands run via sudo using PATH
    runner := system.ShellRunner{}
    if err := system.StartEject(ctx, runner); err != nil {
        // Keep screen displayed; fall through to wait retries
    }

    const timeoutSec = 60
    for {
        err := system.WaitForEject(ctx, runner, timeoutSec)
        if err == nil {
            // Ejected; exit program
            return nil
        }
        // Detect timeout via error text; as we don't have exit status details from Runner yet,
        // retry unconditionally after a short pause.
        if errors.Is(err, context.DeadlineExceeded) {
            // If context timeout, break
            return err
        }
        time.Sleep(500 * time.Millisecond)
    }
}

func (a *App) Stop() error {
    // Stop subsystems in the future; for now no-op
    return nil
}
