package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rook-computer/keymaker/internal/app"
	"github.com/rook-computer/keymaker/internal/buttons"
	"github.com/rook-computer/keymaker/internal/flash"
	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/web"
)

func main() {
	fmt.Println("Keymaker starting (skeleton)")

	// Flags
	debug := flag.Bool("debug", false, "enable debug logging to ./keymaker-debug.log")
	noLogo := flag.Bool("no-logo", false, "disable logo rendering")
	flag.Parse()

	// Local file logger when debug enabled
	var logger app.Logger = app.NoopLogger{}
	if *debug {
		f, err := os.OpenFile("./keymaker-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			logger = app.NewFileLogger(f)
			logger.Infof("main", "debug logging enabled")
		} else {
			fmt.Println("debug log open error:", err)
		}
	}

	// Context for lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Shared state store
	store := state.NewStore()

	// Subsystem stubs (renderer is real to show the local UI)
	renderer := render.NewFBRenderer()
	server := &web.NoopServer{}
	flasher := &flash.NoopFlasher{}
	btns := buttons.NewNoopButtons()

	// App construction
	a := app.New(store, renderer, server, flasher, btns)
	a.Logger = logger
	a.NoLogo = *noLogo
	a.Debug = *debug

	if err := a.Start(ctx); err != nil {
		fmt.Println("app start error:", err)
		return
	}

	// Sleep briefly to simulate lifecycle; real app would block until shutdown
	time.Sleep(100 * time.Millisecond)

	// Stop app
	if err := a.Stop(); err != nil {
		fmt.Println("app stop error:", err)
	}
}
