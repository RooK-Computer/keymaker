package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/web"
)

func main() {
	defaults, err := web.DefaultServerConfigFromEnv(":8080")
	if err != nil {
		fmt.Println("server config error:", err)
		os.Exit(2)
	}

	listenAddr := flag.String("listen", defaults.ListenAddr, "http listen address; also configurable via "+web.EnvListenAddr)
	devMode := flag.Bool("dev", defaults.DevMode, "enable dev mode; also configurable via "+web.EnvDevMode)
	staticDir := flag.String("static-dir", "", "serve static UI from this directory (optional); when empty, embedded web UI assets are served")
	scenario := flag.String("scenario", "retropie", "simulator cartridge scenario: retropie | no-cartridge | unknown")
	cartridgeRoot := flag.String("cartridge-root", "/tmp/keymaker-sim/cartridge", "simulated cartridge root directory")
	flag.Parse()

	// Ensure the simulator starts in a safe default state.
	info := state.GetCartridgeInfo()
	info.Reset()

	processCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := filepath.Clean(*cartridgeRoot)
	startupScenario := strings.TrimSpace(*scenario)
	if startupScenario == "" {
		startupScenario = "retropie"
	}

	control := NewSimControl(processCtx, root, startupScenario, info)
	if err := control.ApplyScenario(startupScenario); err != nil {
		fmt.Println("scenario init error:", err)
		os.Exit(2)
	}

	deps := control.Deps()

	server := web.NewHTTPServer(web.ServerConfig{ListenAddr: *listenAddr, DevMode: *devMode})
	server.StaticDir = *staticDir
	server.Handler = web.NewDefaultMux(server.StaticDir, web.APIV1Config{
		Handlers: web.APIV1Handlers{EjectFunc: control.Eject, FlashFunc: control.Flash},
		Deps:     deps,
	})
	registerSimEndpoints(server.Handler, control)

	if err := server.Start(processCtx); err != nil {
		fmt.Println("server start error:", err)
		os.Exit(1)
	}

	fmt.Println("Keymaker simulator listening on", server.Addr)
	fmt.Println("Scenario:", startupScenario)
	fmt.Println("Cartridge root:", root)
	fmt.Println("API: http://" + trimLeadingColon(server.Addr) + "/api/v1/")

	<-processCtx.Done()
	_ = server.Stop()
}

func trimLeadingColon(addr string) string {
	// Best-effort for display; don't attempt full URL parsing here.
	if len(addr) > 0 && addr[0] == ':' {
		return "127.0.0.1" + addr
	}
	if addr == "" {
		return "127.0.0.1:8080"
	}
	// If it's already a host:port, keep it.
	return addr
}

var _ = http.ErrServerClosed

func applyScenario(ctx context.Context, root, scenario string, info *state.CartridgeInfo) error {
	_ = ctx
	if info == nil {
		info = state.GetCartridgeInfo()
	}
	switch scenario {
	case "no-cartridge":
		info.SetPresent(false)
		info.SetMounted(false)
		info.SetRetroPie(false, nil)
		return nil
	case "unknown":
		if err := os.MkdirAll(root, 0o755); err != nil {
			return err
		}
		info.SetPresent(true)
		info.SetMounted(false)
		info.SetRetroPie(false, nil)
		return nil
	case "retropie", "":
		if err := seedRetroPie(root); err != nil {
			return err
		}
		info.SetPresent(true)
		info.SetMounted(false)
		info.SetRetroPie(true, []string{"nes", "snes"})
		return nil
	default:
		return fmt.Errorf("unknown scenario %q", scenario)
	}
}

func seedRetroPie(root string) error {
	romsRoot := filepath.Join(root, "home/pi/RetroPie/roms")
	if err := os.MkdirAll(filepath.Join(romsRoot, "nes"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(romsRoot, "snes"), 0o755); err != nil {
		return err
	}
	// Seed a couple of tiny dummy files so list/download/upload flows have something to work with.
	if err := os.WriteFile(filepath.Join(romsRoot, "nes", "mario.nes"), []byte("dummy-nes-rom\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(romsRoot, "snes", "zelda.sfc"), []byte("dummy-snes-rom\n"), 0o644); err != nil {
		return err
	}
	return nil
}
