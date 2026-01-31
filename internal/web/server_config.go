package web

import (
	"fmt"
	"os"
	"strconv"
)

const (
	EnvListenAddr = "KEYMAKER_LISTEN"
	EnvDevMode    = "KEYMAKER_DEV"
)

// ServerConfig contains settings for running the HTTP server.
//
// The intended defaults differ per binary:
// - real device: :80
// - simulator:   :8080
type ServerConfig struct {
	ListenAddr string
	DevMode    bool
}

func DefaultServerConfigFromEnv(defaultListenAddr string) (ServerConfig, error) {
	listenAddr := os.Getenv(EnvListenAddr)
	if listenAddr == "" {
		listenAddr = defaultListenAddr
	}

	devMode := false
	if raw := os.Getenv(EnvDevMode); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return ServerConfig{}, fmt.Errorf("%s must be a boolean (got %q): %w", EnvDevMode, raw, err)
		}
		devMode = parsed
	}

	return ServerConfig{ListenAddr: listenAddr, DevMode: devMode}, nil
}
