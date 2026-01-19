package render

import (
	"context"
	"image"
	"image/color"

	"github.com/rook-computer/keymaker/internal/state"
)

type Renderer interface {
	Start(ctx context.Context) error
	Stop() error
	SetScreen(screen Screen)
	RunLoop(ctx context.Context, store *state.Store)
	RedrawWithState(snap state.State)
}

type Screen interface {
	Start(ctx context.Context) error
	Stop() error
	Draw(r Drawer, s state.State)
}

// Stub implementations
type NoopRenderer struct{}

func (n *NoopRenderer) Start(ctx context.Context) error                 { return nil }
func (n *NoopRenderer) Stop() error                                     { return nil }
func (n *NoopRenderer) SetScreen(screen Screen)                         {}
func (n *NoopRenderer) RunLoop(ctx context.Context, store *state.Store) {}
func (n *NoopRenderer) RedrawWithState(snap state.State)                {}

// Drawer is an abstraction the renderer provides to screens to draw primitives
// without exposing low-level framebuffer details.
type Drawer interface {
	// Size returns the logical canvas size (in pixels) that screens draw into.
	Size() (width int, height int)

	FillBackground()

	// Generic text primitives.
	MeasureText(text string, style TextStyle) TextMetrics
	DrawText(text string, x, y int, style TextStyle) TextMetrics

	// Generic image primitives.
	ImageSize(img image.Image) (width int, height int)
	DrawImage(img image.Image, x, y int, opts ImageOpts)
	DrawImageInRect(img image.Image, rect image.Rectangle, mode ScaleMode)

	// Convenience helpers (implemented using the generic primitives).
	DrawLogoCenteredTop()
	DrawTextCentered(text string)
}

type TextAlign int

const (
	TextAlignLeft TextAlign = iota
	TextAlignCenter
	TextAlignRight
)

// TextStyle describes how to render text.
// Coordinates for DrawText use a top-left anchor for Y.
// For X, Align controls how x is interpreted.
type TextStyle struct {
	Color color.Color
	Size  int // font size in points; 0 means renderer default
	Align TextAlign
}

type TextMetrics struct {
	Width      int
	Height     int
	Ascent     int
	Descent    int
	LineHeight int
}

type ScaleMode int

const (
	ScaleModeFit ScaleMode = iota
	ScaleModeFill
	ScaleModeStretch
)

type ImageOpts struct {
	// Reserved for future use (composition mode, opacity, etc.).
}
