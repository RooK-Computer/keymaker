package screens

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rook-computer/keymaker/internal/cartridge"
	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

type InsertCartridgeScreen struct {
	Runner system.Runner
	Logger Logger
	App    AppController

	TimeoutSeconds int

	cancel context.CancelFunc

	mu      sync.RWMutex
	message string
}

func NewInsertCartridgeScreen(runner system.Runner, logger Logger, app AppController) *InsertCartridgeScreen {
	return &InsertCartridgeScreen{
		Runner:         runner,
		Logger:         logger,
		App:            app,
		TimeoutSeconds: 300,
		message:        "please insert cartridge\n(this may take a while...)",
	}
}

func (screen *InsertCartridgeScreen) Start(ctx context.Context) error {
	if screen.Runner == nil {
		return errors.New("no system runner configured")
	}
	if screen.App == nil {
		return errors.New("no app controller configured")
	}

	screenCtx, cancel := context.WithCancel(ctx)
	screen.cancel = cancel

	cartridgeInfo := state.GetCartridgeInfo()
	cartridgeInfo.Reset()
	screen.setMessage("please insert cartridge\n(this may take a while...)")

	go func() {
		for {
			if err := system.WaitForInsert(screenCtx, screen.Runner, screen.TimeoutSeconds); err == nil {
				break
			}
			if screenCtx.Err() != nil {
				return
			}
			if screen.Logger != nil {
				screen.Logger.Infof("app", "still waiting for cartridge (last wait timed out after %ds)", screen.TimeoutSeconds)
			}
		}

		screen.setMessage("analyzing cartridge")
		_ = cartridge.DetectAndUpdate(screenCtx, screen.Runner, screen.Logger, cartridge.DetectOptions{
			HasWorkCartridge: true,
			ManageBusy:       true,
			Retries:          3,
			RetryDelay:       750 * time.Millisecond,
		})
		screen.setMessage("cartridge ready")
		nextScreen := NewWiFiSetupScreen(screen.Runner, screen.Logger, screen.App)
		if err := screen.App.SetScreen(nextScreen); err != nil {
			if screen.Logger != nil {
				screen.Logger.Errorf("app", "failed to switch to wifi setup screen: %v", err)
			}
			screen.App.Exit(err)
			return
		}
	}()

	return nil
}

func (screen *InsertCartridgeScreen) Stop() error {
	if screen.cancel != nil {
		screen.cancel()
	}
	return nil
}

func (screen *InsertCartridgeScreen) setMessage(message string) {
	screen.mu.Lock()
	screen.message = message
	screen.mu.Unlock()
}

func (screen *InsertCartridgeScreen) getMessage() string {
	screen.mu.RLock()
	defer screen.mu.RUnlock()
	return screen.message
}

func (screen *InsertCartridgeScreen) Draw(drawer render.Drawer, currentState state.State) {
	drawer.FillBackground()
	drawer.DrawLogoCenteredTop()
	drawer.DrawTextCentered(screen.getMessage())
}
