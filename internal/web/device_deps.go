package web

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

const deviceRetroPieRomsRoot = "/cartridge/home/pi/RetroPie/roms"

// NewDeviceAPIV1Deps wires the API to the real device behaviors.
//
// Note: this uses OS scripts for mounting, so it must never be used by the simulator.
func NewDeviceAPIV1Deps(logger sysLogger) APIV1Deps {
	cartridge := state.GetCartridgeInfo()
	if logger == nil {
		logger = noopSysLogger{}
	}
	return APIV1Deps{
		Cartridge: cartridge,
		Mounter:   DeviceCartridgeMounter{Cartridge: cartridge, Logger: logger},
		RetroPie:  FileSystemRetroPieStorage{RomsRoot: deviceRetroPieRomsRoot},
	}
}

type DeviceCartridgeMounter struct {
	Cartridge CartridgeInfoStore
	Logger    sysLogger
}

type noopSysLogger struct{}

func (noopSysLogger) Infof(string, string, ...interface{})  {}
func (noopSysLogger) Errorf(string, string, ...interface{}) {}

func (m DeviceCartridgeMounter) EnsureMounted(ctx context.Context) error {
	if m.Cartridge == nil {
		m.Cartridge = state.GetCartridgeInfo()
	}
	if m.Logger == nil {
		m.Logger = noopSysLogger{}
	}
	if m.Cartridge.Snapshot().Mounted {
		return nil
	}

	runner := system.ShellRunner{Logger: m.Logger}
	if err := system.MountCartridge(ctx, runner); err != nil {
		return err
	}
	// Give the kernel a brief moment to settle on slow cards.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	m.Cartridge.SetMounted(true)
	return nil
}

type FileSystemRetroPieStorage struct {
	RomsRoot string
}

func (s FileSystemRetroPieStorage) ListGames(ctx context.Context, systemName string) ([]string, error) {
	_ = ctx
	return listGamesForSystem(s.RomsRoot, systemName)
}

func (s FileSystemRetroPieStorage) DownloadGame(ctx context.Context, w http.ResponseWriter, r *http.Request, systemName, gameName string) error {
	_ = ctx
	return downloadGame(s.RomsRoot, w, r, systemName, gameName)
}

func (s FileSystemRetroPieStorage) UploadGame(ctx context.Context, systemName, gameName string, body io.Reader, contentLength int64) error {
	_ = ctx
	return uploadGame(s.RomsRoot, systemName, gameName, body, contentLength)
}

func (s FileSystemRetroPieStorage) DeleteGame(ctx context.Context, systemName, gameName string) error {
	_ = ctx
	return deleteGame(s.RomsRoot, systemName, gameName)
}
