package app

import (
    "context"
    "time"
    "sync"
    "io"
    "fmt"
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
    Logger  Logger
    NoLogo  bool
    Debug   bool
}

func New(store *state.Store, r render.Renderer, w web.Server, f flash.Flasher, b buttons.Buttons) *App {
    return &App{Store: store, Render: r, Web: w, Flash: f, Buttons: b, Logger: NoopLogger{}}
}

func (a *App) Start(ctx context.Context) error {
    a.Store.SetPhase(state.READY)
    // Initialize renderer and draw first screen
    fb := render.NewFBRenderer()
    fb.Logger = a.Logger
    fb.NoLogo = a.NoLogo
    fb.Debug = a.Debug
    if err := fb.Start(ctx); err != nil { a.Logger.Errorf("app", "fb start error: %v", err); return err }
    defer fb.Stop()
    // Switch console to KD_GRAPHICS to suppress hardware cursor
    if err := system.SetGraphicsModeWithLog(a.Logger); err != nil { a.Logger.Errorf("tty", "set graphics mode failed: %v", err) }
    _ = system.HideCursorWithLog(a.Logger)
    defer func(){ _ = system.ShowCursorWithLog(a.Logger); _ = system.RestoreTextModeWithLog(a.Logger) }()
    fb.SetScreen(render.RemoveCartridgeScreen{})
    // Force immediate first redraw to ensure text shows without waiting for loop
    fb.RedrawWithState(a.Store.Snapshot())
    // Start render loop so the framebuffer refreshes and covers any blinking cursor
    loopCtx, cancel := context.WithCancel(ctx)
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        fb.RunLoop(loopCtx, a.Store)
    }()

    // Begin ejection process and wait with timeout retries
    // Use ShellRunner so commands run via sudo using PATH
    runner := system.ShellRunner{Logger: a.Logger}
    if err := system.StartEject(ctx, runner); err != nil {
        // Keep screen displayed; fall through to wait retries
        a.Logger.Errorf("system", "start eject failed: %v", err)
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
            select {
            case <-loopCtx.Done():
                return
            case <-time.After(500 * time.Millisecond):
                a.Logger.Infof("app", "waiting for eject...")
            }
        }
    }()

    // Wait for completion
    err := <-done
    cancel()
    wg.Wait()
    return err
}

func (a *App) Stop() error {
    // Stop subsystems in the future; for now no-op
    return nil
}

// Logger interface and implementations
type Logger interface {
    Infof(component string, format string, args ...interface{})
    Errorf(component string, format string, args ...interface{})
}

type NoopLogger struct{}
func (NoopLogger) Infof(component, format string, args ...interface{}) {}
func (NoopLogger) Errorf(component, format string, args ...interface{}) {}

type FileLogger struct{ w io.Writer }
func NewFileLogger(w io.Writer) FileLogger { return FileLogger{w: w} }
func (l FileLogger) Infof(component string, format string, args ...interface{}) { writeLog(l.w, "INFO", component, format, args...) }
func (l FileLogger) Errorf(component string, format string, args ...interface{}) { writeLog(l.w, "ERROR", component, format, args...) }

func writeLog(w io.Writer, level, component, format string, args ...interface{}) {
    ts := time.Now().Format(time.RFC3339)
    msg := fmt.Sprintf(format, args...)
    _, _ = io.WriteString(w, ts+" ["+level+"] "+component+": "+msg+"\n")
}
