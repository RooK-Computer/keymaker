package app

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rook-computer/keymaker/internal/app/screens"
	"github.com/rook-computer/keymaker/internal/buttons"
	"github.com/rook-computer/keymaker/internal/flash"
	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
	"github.com/rook-computer/keymaker/internal/web"
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

	currentScreen render.Screen

	exitOnce atomic.Bool
	exitCh   chan error
}

func New(store *state.Store, r render.Renderer, w web.Server, f flash.Flasher, b buttons.Buttons) *App {
	return &App{Store: store, Render: r, Web: w, Flash: f, Buttons: b, Logger: NoopLogger{}, exitCh: make(chan error, 1)}
}

// Exit requests the app to stop running.
// Any screen can call this to terminate the process via the generic codepath.
func (a *App) Exit(err error) {
	if a.exitCh == nil {
		return
	}
	if !a.exitOnce.CompareAndSwap(false, true) {
		return
	}
	select {
	case a.exitCh <- err:
	default:
	}
}

func (a *App) Start(ctx context.Context) error {
	if a.exitCh == nil {
		a.exitCh = make(chan error, 1)
	}
	a.exitOnce.Store(false)

	a.Store.SetPhase(state.READY)
	// Initialize renderer and draw first screen
	if a.Render == nil {
		a.Render = render.NewFBRenderer()
	}
	if fb, ok := a.Render.(*render.FBRenderer); ok {
		fb.Logger = a.Logger
		fb.NoLogo = a.NoLogo
		fb.Debug = a.Debug
	}
	if err := a.Render.Start(ctx); err != nil {
		a.Logger.Errorf("app", "renderer start error: %v", err)
		return err
	}
	defer a.Render.Stop()

	// Switch console to KD_GRAPHICS to suppress hardware cursor
	if err := system.SetGraphicsModeWithLog(a.Logger); err != nil {
		a.Logger.Errorf("tty", "set graphics mode failed: %v", err)
	}
	_ = system.HideCursorWithLog(a.Logger)
	defer func() { _ = system.ShowCursorWithLog(a.Logger); _ = system.RestoreTextModeWithLog(a.Logger) }()

	// Show the ejection screen; it owns the eject/wait logic.
	runner := system.ShellRunner{Logger: a.Logger}
	ejectScreen := screens.NewRemoveCartridgeScreen(runner, a.Logger, a)
	if err := a.setScreen(ctx, ejectScreen); err != nil {
		return err
	}

	// Force immediate first redraw to ensure text shows without waiting for loop.
	a.Render.RedrawWithState(a.Store.Snapshot())

	// Start render loop so the framebuffer refreshes and covers any blinking cursor.
	loopCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.Render.RunLoop(loopCtx, a.Store)
	}()

	// Wait for completion (requested by a screen), then exit.
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-a.exitCh:
	}
	cancel()
	wg.Wait()
	return err
}

func (a *App) setScreen(ctx context.Context, s render.Screen) error {
	if a.currentScreen != nil {
		_ = a.currentScreen.Stop()
	}
	a.currentScreen = s
	a.Render.SetScreen(s)
	return s.Start(ctx)
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

func (NoopLogger) Infof(component, format string, args ...interface{})  {}
func (NoopLogger) Errorf(component, format string, args ...interface{}) {}

type FileLogger struct{ w io.Writer }

func NewFileLogger(w io.Writer) FileLogger { return FileLogger{w: w} }
func (l FileLogger) Infof(component string, format string, args ...interface{}) {
	writeLog(l.w, "INFO", component, format, args...)
}
func (l FileLogger) Errorf(component string, format string, args ...interface{}) {
	writeLog(l.w, "ERROR", component, format, args...)
}

func writeLog(w io.Writer, level, component, format string, args ...interface{}) {
	ts := time.Now().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	_, _ = io.WriteString(w, ts+" ["+level+"] "+component+": "+msg+"\n")
}
