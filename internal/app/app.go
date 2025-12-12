package app

import (
    "context"
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
    // Switch console to KD_GRAPHICS to suppress hardware cursor
    _ = system.SetGraphicsMode()
    defer system.RestoreTextMode()
    fb.SetScreen(render.RemoveCartridgeScreen{})
    // Start render loop so the framebuffer refreshes and covers any blinking cursor
    loopCtx, cancel := context.WithCancel(ctx)
    go fb.RunLoop(loopCtx, a.Store)

    // Begin ejection process and wait with timeout retries
    // Use ShellRunner so commands run via sudo using PATH
    runner := system.ShellRunner{}
    if err := system.StartEject(ctx, runner); err != nil {
        // Keep screen displayed; fall through to wait retries
    }

    // Run eject sequence in a separate goroutine
    done := make(chan error, 1)
    go func() {
        _ = system.StartEject(ctx, runner)
        const timeoutSec = 60
        for {
            if err := system.WaitForEject(ctx, runner, timeoutSec); err == nil {
                done <- nil
                return
            }
            // retry after short pause
            time.Sleep(500 * time.Millisecond)
        }
    }()

    // Wait for completion
    err := <-done
    cancel()
    return err
}

func (a *App) Stop() error {
    // Stop subsystems in the future; for now no-op
    return nil
}
