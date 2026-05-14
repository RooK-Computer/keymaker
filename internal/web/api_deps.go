package web

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/rook-computer/keymaker/internal/state"
)

// CartridgeInfoStore abstracts the cartridge state used by the API.
//
// The concrete implementation is typically state.GetCartridgeInfo().
type CartridgeInfoStore interface {
	Snapshot() state.CartridgeInfoSnapshot
	SetMounted(mounted bool)
}

// sysLogger matches the logging shape used by system.ShellRunner.
// It is intentionally tiny so callers can pass existing loggers without adapters.
type sysLogger interface {
	Infof(component string, format string, args ...interface{})
	Errorf(component string, format string, args ...interface{})
}

// CartridgeMounter abstracts mounting behavior.
//
// Device implementations may use OS scripts; simulator implementations must not.
type CartridgeMounter interface {
	EnsureMounted(ctx context.Context) error
}

// RetroPieStorage abstracts file operations for the RetroPie roms tree.
type RetroPieStorage interface {
	ListGames(ctx context.Context, systemName string) ([]string, error)
	DownloadGame(ctx context.Context, w http.ResponseWriter, r *http.Request, systemName, gameName string) error
	UploadGame(ctx context.Context, systemName, gameName string, body io.Reader, contentLength int64) error
	DeleteGame(ctx context.Context, systemName, gameName string) error
}

// VisibleWiFiNetworks lists the SSIDs currently known to the application.
type VisibleWiFiNetworks interface {
	ListVisibleWiFiNetworks(ctx context.Context) ([]string, error)
}

// WiFiJoiner requests that the device switch to another WiFi network.
type WiFiJoiner interface {
	RequestWiFiJoin(ctx context.Context, ssid, password string) error
}

type APIV1Deps struct {
	Cartridge           CartridgeInfoStore
	Mounter             CartridgeMounter
	RetroPie            RetroPieStorage
	VisibleWiFiNetworks VisibleWiFiNetworks
	WiFiJoiner          WiFiJoiner
}

func (d APIV1Deps) withDefaults() APIV1Deps {
	out := d
	if out.Cartridge == nil {
		out.Cartridge = state.GetCartridgeInfo()
	}
	if out.Mounter == nil {
		out.Mounter = NoopCartridgeMounter{Err: errors.New("mount not configured")}
	}
	if out.RetroPie == nil {
		out.RetroPie = NoopRetroPieStorage{Err: errors.New("retropie storage not configured")}
	}
	if out.VisibleWiFiNetworks == nil {
		out.VisibleWiFiNetworks = NoopVisibleWiFiNetworks{Err: errors.New("visible wifi networks not configured")}
	}
	if out.WiFiJoiner == nil {
		out.WiFiJoiner = NoopWiFiJoiner{Err: errors.New("wifi join not configured")}
	}
	return out
}

type NoopCartridgeMounter struct{ Err error }

func (m NoopCartridgeMounter) EnsureMounted(context.Context) error {
	if m.Err != nil {
		return m.Err
	}
	return errors.New("mount not configured")
}

type NoopRetroPieStorage struct{ Err error }

func (s NoopRetroPieStorage) ListGames(context.Context, string) ([]string, error) {
	return nil, s.err()
}

func (s NoopRetroPieStorage) DownloadGame(context.Context, http.ResponseWriter, *http.Request, string, string) error {
	return s.err()
}

func (s NoopRetroPieStorage) UploadGame(context.Context, string, string, io.Reader, int64) error {
	return s.err()
}

func (s NoopRetroPieStorage) DeleteGame(context.Context, string, string) error {
	return s.err()
}

func (s NoopRetroPieStorage) err() error {
	if s.Err != nil {
		return s.Err
	}
	return errors.New("retropie storage not configured")
}

type NoopVisibleWiFiNetworks struct{ Err error }

func (s NoopVisibleWiFiNetworks) ListVisibleWiFiNetworks(context.Context) ([]string, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return nil, errors.New("visible wifi networks not configured")
}

type NoopWiFiJoiner struct{ Err error }

func (j NoopWiFiJoiner) RequestWiFiJoin(context.Context, string, string) error {
	if j.Err != nil {
		return j.Err
	}
	return errors.New("wifi join not configured")
}
