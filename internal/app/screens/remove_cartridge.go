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

func (s *RemoveCartridgeScreen) Start(ctx context.Context) error {
	if s.Runner == nil {
		return errors.New("no system runner configured")
	}
	if s.Exiter == nil {
		return errors.New("no app exiter configured")
	}

	screenCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go func() {
		// Enable lifeline before ejection (best-effort).
		if err := system.LifelineOn(screenCtx, s.Runner); err != nil {
			if s.Logger != nil {
				s.Logger.Errorf("system", "lifeline on failed: %v", err)
			}
		}

		if err := system.StartEject(screenCtx, s.Runner); err != nil {
			if s.Logger != nil {
				s.Logger.Errorf("system", "start eject failed: %v", err)
			}
			// Keep going: the wait loop will still retry and may succeed.
		}

		for {
			if err := system.WaitForEject(screenCtx, s.Runner, s.TimeoutSeconds); err == nil {
				s.Exiter.Exit(nil)
				return
			}

			select {
			case <-screenCtx.Done():
				return
			case <-time.After(s.RetryDelay):
				if s.Logger != nil {
					s.Logger.Infof("app", "waiting for eject...")
				}
			}
		}
	}()

	return nil
}

func (s *RemoveCartridgeScreen) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *RemoveCartridgeScreen) Draw(r render.Drawer, st state.State) {
	r.FillBackground()
	r.DrawLogoCenteredTop()
	r.DrawTextCentered("please remove cartridge")
}
