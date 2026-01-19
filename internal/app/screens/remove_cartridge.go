package screens

import (
	"context"
	"errors"
	"time"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

type Logger interface {
	Infof(component string, format string, args ...interface{})
	Errorf(component string, format string, args ...interface{})
}

// AppExiter is implemented by the host application.
// Screens can call Exit to request termination.
type AppExiter interface {
	Exit(err error)
}

// RemoveCartridgeScreen is the ejection screen.
// It initiates SD ejection and waits until the kernel reports removal.
type RemoveCartridgeScreen struct {
	Runner system.Runner
	Logger Logger
	Exiter AppExiter

	TimeoutSeconds int
	RetryDelay     time.Duration

	cancel context.CancelFunc
}

func NewRemoveCartridgeScreen(runner system.Runner, logger Logger, exiter AppExiter) *RemoveCartridgeScreen {
	return &RemoveCartridgeScreen{
		Runner:         runner,
		Logger:         logger,
		Exiter:         exiter,
		TimeoutSeconds: 60,
		RetryDelay:     500 * time.Millisecond,
	}
}

func (screen *RemoveCartridgeScreen) Start(ctx context.Context) error {
	if screen.Runner == nil {
		return errors.New("no system runner configured")
	}
	if screen.Exiter == nil {
		return errors.New("no app exiter configured")
	}

	screenCtx, cancel := context.WithCancel(ctx)
	screen.cancel = cancel

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

		for {
			if err := system.WaitForEject(screenCtx, screen.Runner, screen.TimeoutSeconds); err == nil {
				screen.Exiter.Exit(nil)
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

func (screen *RemoveCartridgeScreen) Draw(drawer render.Drawer, currentState state.State) {
	drawer.FillBackground()
	drawer.DrawLogoCenteredTop()
	drawer.DrawTextCentered("please remove cartridge")
}
