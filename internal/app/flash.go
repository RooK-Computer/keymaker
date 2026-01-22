package app

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/rook-computer/keymaker/internal/cartridge"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

// HandleFlash is used by the web API to overwrite the cartridge with a gzipped disk image.
// It must not buffer the input; it streams into the flashing pipeline.
func (app *App) HandleFlash(ctx context.Context, reader io.Reader) error {
	if app.Flash == nil {
		return errors.New("flasher not configured")
	}

	state.GetCartridgeInfo().SetBusy(true)
	defer state.GetCartridgeInfo().SetBusy(false)

	// Ensure unmounted before dd.
	runner := system.ShellRunner{Logger: app.Logger}
	snap := state.GetCartridgeInfo().Snapshot()
	if snap.Mounted {
		if err := system.UnmountCartridge(ctx, runner); err != nil {
			return err
		}
		state.GetCartridgeInfo().SetMounted(false)
	}

	// Cartridge content will change; clear cached type/systems.
	state.GetCartridgeInfo().SetRetroPie(false, nil)

	if app.Store != nil {
		app.Store.SetPhase(state.FLASHING)
		app.Store.UpdateFlash(state.FlashInfo{Status: "flashing"})
	}

	err := app.Flash.Start(ctx, reader)
	if err == nil {
		// Re-detect cartridge contents after flashing (partitions may take a moment to settle).
		_ = cartridge.DetectAndUpdate(ctx, runner, app.Logger, cartridge.DetectOptions{
			HasWorkCartridge: true,
			ManageBusy:       false,
			Retries:          6,
			RetryDelay:       1 * time.Second,
		})
	}
	if app.Store != nil {
		if err != nil {
			app.Store.SetPhase(state.ERROR)
			app.Store.UpdateFlash(state.FlashInfo{Status: "error", Err: err.Error()})
		} else {
			app.Store.SetPhase(state.DONE)
			app.Store.UpdateFlash(state.FlashInfo{Status: "done"})
		}
	}
	return err
}
