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

func New(store *state.Store, renderer render.Renderer, webServer web.Server, flasher flash.Flasher, buttonDriver buttons.Buttons) *App {
	return &App{Store: store, Render: renderer, Web: webServer, Flash: flasher, Buttons: buttonDriver, Logger: NoopLogger{}, exitCh: make(chan error, 1)}
}

// Exit requests the app to stop running.
// Any screen can call this to terminate the process via the generic codepath.
func (app *App) Exit(err error) {
	if app.exitCh == nil {
		return
	}
	if !app.exitOnce.CompareAndSwap(false, true) {
		return
	}
	select {
	case app.exitCh <- err:
	default:
	}
}

func (app *App) Start(ctx context.Context) error {
	if app.exitCh == nil {
		app.exitCh = make(chan error, 1)
	}
	app.exitOnce.Store(false)

	app.Store.SetPhase(state.READY)
	// Initialize renderer and draw first screen
	if app.Render == nil {
		app.Render = render.NewFBRenderer()
	}
	if fb, ok := app.Render.(*render.FBRenderer); ok {
		fb.Logger = app.Logger
		fb.NoLogo = app.NoLogo
		fb.Debug = app.Debug
	}
	if err := app.Render.Start(ctx); err != nil {
		app.Logger.Errorf("app", "renderer start error: %v", err)
		return err
	}
	defer app.Render.Stop()

	// Switch console to KD_GRAPHICS to suppress hardware cursor
	if err := system.SetGraphicsModeWithLog(app.Logger); err != nil {
		app.Logger.Errorf("tty", "set graphics mode failed: %v", err)
	}
	_ = system.HideCursorWithLog(app.Logger)
	defer func() { _ = system.ShowCursorWithLog(app.Logger); _ = system.RestoreTextModeWithLog(app.Logger) }()

	// Show the ejection screen; it owns the eject/wait logic.
	runner := system.ShellRunner{Logger: app.Logger}
	ejectScreen := screens.NewRemoveCartridgeScreen(runner, app.Logger, app)
	if err := app.setScreen(ctx, ejectScreen); err != nil {
		return err
	}

	// Force immediate first redraw to ensure text shows without waiting for loop.
	app.Render.RedrawWithState(app.Store.Snapshot())

	// Start render loop so the framebuffer refreshes and covers any blinking cursor.
	loopCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.Render.RunLoop(loopCtx, app.Store)
	}()

	// Wait for completion (requested by a screen), then exit.
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-app.exitCh:
	}
	cancel()
	wg.Wait()
	return err
}

func (app *App) setScreen(ctx context.Context, screen render.Screen) error {
	if app.currentScreen != nil {
		_ = app.currentScreen.Stop()
	}
	app.currentScreen = screen
	app.Render.SetScreen(screen)
	return screen.Start(ctx)
}

func (app *App) Stop() error {
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
	timestamp := time.Now().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	_, _ = io.WriteString(w, timestamp+" ["+level+"] "+component+": "+msg+"\n")
}
