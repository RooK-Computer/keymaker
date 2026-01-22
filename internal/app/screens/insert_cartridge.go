package screens

import (
	"context"
	"errors"
	"sync"

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
