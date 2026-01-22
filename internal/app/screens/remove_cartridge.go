package screens

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

type Logger interface {
	Infof(component string, format string, args ...interface{})
	Errorf(component string, format string, args ...interface{})
}

// AppController is implemented by the host application.
// Screens can switch screens and (on error) request termination.
type AppController interface {
	SetScreen(screen render.Screen) error
	Exit(err error)
}

// RemoveCartridgeScreen is the ejection screen.
// It initiates SD ejection and waits until the kernel reports removal.
type RemoveCartridgeScreen struct {
	Runner system.Runner
	Logger Logger
	App    AppController

	TimeoutSeconds int
	RetryDelay     time.Duration

	cancel context.CancelFunc

	mu      sync.RWMutex
	message string
}

func NewRemoveCartridgeScreen(runner system.Runner, logger Logger, app AppController) *RemoveCartridgeScreen {
	return &RemoveCartridgeScreen{
		Runner:         runner,
		Logger:         logger,
		App:            app,
		TimeoutSeconds: 60,
		RetryDelay:     500 * time.Millisecond,
		message:        "preparing the system...",
	}
}

func (screen *RemoveCartridgeScreen) Start(ctx context.Context) error {
	if screen.Runner == nil {
		return errors.New("no system runner configured")
	}
	if screen.App == nil {
		return errors.New("no app controller configured")
	}

	screenCtx, cancel := context.WithCancel(ctx)
	screen.cancel = cancel

	screen.setMessage("preparing the system...")

	go func() {
		// Enable lifeline before ejection (best-effort).
		if err := system.LifelineOn(screenCtx, screen.Runner); err != nil {
			if screen.Logger != nil {
				screen.Logger.Errorf("system", "lifeline on failed: %v", err)
			}
		}

		if err := system.StartEject(screenCtx, screen.Runner); err != nil {
			if screen.Logger != nil {
				screen.Logger.Errorf("system", "start eject failed: %v", err)
			}
			// Keep going: the wait loop will still retry and may succeed.
		}

		screen.setMessage("please remove cartridge")

		for {
			if err := system.WaitForEject(screenCtx, screen.Runner, screen.TimeoutSeconds); err == nil {
				nextScreen := NewInsertCartridgeScreen(screen.Runner, screen.Logger, screen.App)
				if err := screen.App.SetScreen(nextScreen); err != nil {
					if screen.Logger != nil {
						screen.Logger.Errorf("app", "failed to switch to insert cartridge screen: %v", err)
					}
					screen.App.Exit(err)
					return
				}
				return
			}

			select {
			case <-screenCtx.Done():
				return
			case <-time.After(screen.RetryDelay):
				if screen.Logger != nil {
					screen.Logger.Infof("app", "waiting for eject...")
				}
			}
		}
	}()

	return nil
}

func (screen *RemoveCartridgeScreen) Stop() error {
	if screen.cancel != nil {
		screen.cancel()
	}
	return nil
}

func (screen *RemoveCartridgeScreen) setMessage(message string) {
	screen.mu.Lock()
	screen.message = message
	screen.mu.Unlock()
}

func (screen *RemoveCartridgeScreen) getMessage() string {
	screen.mu.RLock()
	defer screen.mu.RUnlock()
	return screen.message
}

func (screen *RemoveCartridgeScreen) Draw(drawer render.Drawer, currentState state.State) {
	drawer.FillBackground()
	drawer.DrawLogoCenteredTop()
	drawer.DrawTextCentered(screen.getMessage())
}
