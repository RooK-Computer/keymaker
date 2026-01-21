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

type InsertCartridgeScreen struct {
	Runner system.Runner
	Logger Logger
	App    AppController

	TimeoutSeconds int
	RetryDelay     time.Duration

	cancel context.CancelFunc

	mu      sync.RWMutex
	message string
}

func NewInsertCartridgeScreen(runner system.Runner, logger Logger, app AppController) *InsertCartridgeScreen {
	return &InsertCartridgeScreen{
		Runner:         runner,
		Logger:         logger,
		App:            app,
		TimeoutSeconds: 60,
		RetryDelay:     500 * time.Millisecond,
		message:        "please insert cartridge",
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
	screen.setMessage("please insert cartridge")

	go func() {
		for {
			if err := system.WaitForInsert(screenCtx, screen.Runner, screen.TimeoutSeconds); err == nil {
				break
			}

			select {
			case <-screenCtx.Done():
				return
			case <-time.After(screen.RetryDelay):
				if screen.Logger != nil {
					screen.Logger.Infof("app", "waiting for cartridge...")
				}
			}
		}

		cartridgeInfo.SetPresent(true)
		cartridgeInfo.SetHasWorkCartridge(true)

		screen.setMessage("analyzing cartridge")
		cartridgeInfo.SetBusy(true)

		mountedBefore, err := system.IsCartridgeMounted(screenCtx, screen.Runner)
		if err != nil {
			if screen.Logger != nil {
				screen.Logger.Errorf("system", "mount detection failed: %v", err)
			}
		}

		mountedNow := mountedBefore
		if !mountedBefore {
			if err := system.MountCartridge(screenCtx, screen.Runner); err != nil {
				if screen.Logger != nil {
					screen.Logger.Errorf("system", "mount failed: %v", err)
				}
			} else {
				mountedNow = true
			}
		}
		cartridgeInfo.SetMounted(mountedNow)

		isRetroPie, err := system.IsRetroPieCartridge(screenCtx, screen.Runner)
		if err != nil {
			if screen.Logger != nil {
				screen.Logger.Errorf("system", "retropie check failed: %v", err)
			}
			isRetroPie = false
		}

		var systems []string
		if isRetroPie {
			systems, err = system.RetroPieSystems(screenCtx, screen.Runner)
			if err != nil {
				// Per implementation plan: if systems fail, overrule and treat as not RetroPie.
				if screen.Logger != nil {
					screen.Logger.Errorf("system", "retropie systems failed, treating as non-retropie: %v", err)
				}
				isRetroPie = false
				systems = nil
			}
		}
		cartridgeInfo.SetRetroPie(isRetroPie, systems)

		// If the cartridge wasn't mounted before this screen, ensure it isn't left mounted.
		if !mountedBefore {
			mountedAfter, err := system.IsCartridgeMounted(screenCtx, screen.Runner)
			if err != nil {
				if screen.Logger != nil {
					screen.Logger.Errorf("system", "mount detection failed (post-analyze): %v", err)
				}
			} else if mountedAfter {
				if err := system.UnmountCartridge(screenCtx, screen.Runner); err != nil {
					if screen.Logger != nil {
						screen.Logger.Errorf("system", "unmount failed: %v", err)
					}
				} else {
					cartridgeInfo.SetMounted(false)
				}
			}
		}

		cartridgeInfo.SetBusy(false)
		screen.setMessage("cartridge ready")
		screen.App.Exit(nil)
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
