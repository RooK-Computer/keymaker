package screens

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

type WiFiSetupScreen struct {
	Runner system.Runner
	Logger Logger
	App    AppController

	cancel context.CancelFunc

	mu      sync.RWMutex
	message string
}

func NewWiFiSetupScreen(runner system.Runner, logger Logger, app AppController) *WiFiSetupScreen {
	return &WiFiSetupScreen{
		Runner:  runner,
		Logger:  logger,
		App:     app,
		message: "setting up wifi network",
	}
}

func (screen *WiFiSetupScreen) Start(ctx context.Context) error {
	if screen.Runner == nil {
		return errors.New("no system runner configured")
	}
	if screen.App == nil {
		return errors.New("no app controller configured")
	}

	screenCtx, cancel := context.WithCancel(ctx)
	screen.cancel = cancel

	screen.setMessage("setting up wifi network")

	go func() {
		wifiConfig := state.GetWiFiConfig()
		snapshot := wifiConfig.Snapshot()

		// If WiFi is already configured in our in-memory state, do not attempt
		// to reconfigure/restart WiFi here. This screen is shown after cartridge
		// insertion; during eject flows we want to proceed immediately.
		configured := snapshot.Initialized && (snapshot.Mode == state.WiFiModeHotspot || (snapshot.Mode == state.WiFiModeJoin && strings.TrimSpace(snapshot.SSID) != ""))
		if configured && !snapshot.NeedsApply {
			screen.setMessage("wifi already configured")
			nextScreen := &MainScreen{}
			if err := screen.App.SetScreen(nextScreen); err != nil {
				if screen.Logger != nil {
					screen.Logger.Errorf("app", "failed to switch to main screen: %v", err)
				}
				screen.App.Exit(err)
				return
			}
			return
		}
		networkAvailable := false

		// Unknown state on boot: if any network is available, do nothing and exit.
		if !snapshot.Initialized || snapshot.Mode == state.WiFiModeUnknown {
			wifiIP, wifiErr := system.WiFiIPv4(screenCtx, screen.Runner)
			if wifiErr != nil {
				if screen.Logger != nil {
					screen.Logger.Errorf("system", "netinfo wifi-ip failed: %v", wifiErr)
				}
			}

			ethernetIP, ethernetErr := system.EthernetIPv4(screenCtx, screen.Runner)
			if ethernetErr != nil {
				if screen.Logger != nil {
					screen.Logger.Errorf("system", "netinfo ethernet-ip failed: %v", ethernetErr)
				}
			}

			networkAvailable = wifiIP != "" || ethernetIP != ""
			if !networkAvailable {
				wifiConfig.SetHotspot()
				snapshot = wifiConfig.Snapshot()
			}
		}

		if !networkAvailable {
			switch snapshot.Mode {
			case state.WiFiModeHotspot:
				if err := system.EnsureHotspot(screenCtx, screen.Runner); err != nil {
					if screen.Logger != nil {
						screen.Logger.Errorf("system", "wifi setup failed: %v", err)
					}
					screen.App.Exit(err)
					return
				}
				wifiConfig.MarkApplied()
			case state.WiFiModeJoin:
				if err := system.JoinWiFi(screenCtx, screen.Runner, snapshot.SSID, snapshot.Password); err != nil {
					if screen.Logger != nil {
						screen.Logger.Errorf("system", "wifi setup failed: %v", err)
					}
					screen.App.Exit(err)
					return
				}
				wifiConfig.MarkApplied()
			default:
				// Safety fallback: if mode is still unknown, attempt hotspot.
				wifiConfig.SetHotspot()
				if err := system.EnsureHotspot(screenCtx, screen.Runner); err != nil {
					if screen.Logger != nil {
						screen.Logger.Errorf("system", "wifi setup failed: %v", err)
					}
					screen.App.Exit(err)
					return
				}
				wifiConfig.MarkApplied()
			}
		}

		nextScreen := &MainScreen{}
		if err := screen.App.SetScreen(nextScreen); err != nil {
			if screen.Logger != nil {
				screen.Logger.Errorf("app", "failed to switch to main screen: %v", err)
			}
			screen.App.Exit(err)
			return
		}
	}()

	return nil
}

func (screen *WiFiSetupScreen) Stop() error {
	if screen.cancel != nil {
		screen.cancel()
	}
	return nil
}

func (screen *WiFiSetupScreen) setMessage(message string) {
	screen.mu.Lock()
	screen.message = message
	screen.mu.Unlock()
}

func (screen *WiFiSetupScreen) getMessage() string {
	screen.mu.RLock()
	defer screen.mu.RUnlock()
	return screen.message
}

func (screen *WiFiSetupScreen) Draw(drawer render.Drawer, currentState state.State) {
	drawer.FillBackground()
	drawer.DrawLogoCenteredTop()
	drawer.DrawTextCentered(screen.getMessage())
}
