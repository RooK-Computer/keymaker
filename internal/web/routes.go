package web

import (
	"context"
	"io"
	"net/http"
)

type APIV1Handlers struct {
	EjectFunc func(ctx context.Context) error
	FlashFunc func(ctx context.Context, reader io.Reader) error
}

type APIV1Config struct {
	Handlers APIV1Handlers
	Deps     APIV1Deps
}

// RegisterAPIV1 registers the public API routes under /api/v1/.
func RegisterAPIV1(mux *http.ServeMux, cfg APIV1Config) {
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1RouterWithDeps(cfg.Handlers, cfg.Deps)))
}

// RegisterUI serves either embedded UI assets or a directory.
func RegisterUI(mux *http.ServeMux, staticDir string) {
	mux.Handle("/", StaticUIHandler(staticDir))
}

// NewDefaultMux builds the standard mux used by both the device and simulator:
// - /api/v1/* for the API
// - / for the web UI
func NewDefaultMux(staticDir string, cfg APIV1Config) *http.ServeMux {
	mux := http.NewServeMux()
	RegisterAPIV1(mux, cfg)
	RegisterUI(mux, staticDir)
	return mux
}
