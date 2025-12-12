package main

import (
	"context"
	"fmt"
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

	// Context for lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Shared state store
	store := state.NewStore()

	// No-op subsystem stubs
	renderer := &render.NoopRenderer{}
	server := &web.NoopServer{}
	flasher := &flash.NoopFlasher{}
	btns := buttons.NewNoopButtons()

	// App construction
	a := app.New(store, renderer, server, flasher, btns)

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
