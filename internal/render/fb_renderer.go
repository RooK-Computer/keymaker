package render

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync/atomic"
	"time"

	"github.com/golang/freetype/truetype"
	fb "github.com/gonutz/framebuffer"
	"github.com/rook-computer/keymaker/internal/assets"
	"github.com/rook-computer/keymaker/internal/state"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// FBRenderer renders to the Linux framebuffer using an offscreen logical canvas.
type FBRenderer struct {
	fbDev        *fb.Device
	canvas       *image.RGBA
	fontFace     font.Face
	ttFont       *truetype.Font
	logo         image.Image
	running      atomic.Bool
	current      Screen
	lastLogoRect image.Rectangle
	Logger       interface {
		Infof(string, string, ...interface{})
		Errorf(string, string, ...interface{})
	}
	NoLogo bool
	Debug  bool
}

func NewFBRenderer() *FBRenderer { return &FBRenderer{} }

func (r *FBRenderer) Start(ctx context.Context) error {
	// Open framebuffer
	dev, err := fb.Open("/dev/fb0")
	if err != nil {
		return err
	}
	r.fbDev = dev
	if r.Logger != nil {
		b := dev.Bounds()
		r.Logger.Infof("fb", "framebuffer open, bounds=%dx%d", b.Dx(), b.Dy())
	}

	// Prepare logical canvas
	r.canvas = image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))

	// Load font from embedded bytes
	fnt, err := opentype.Parse(assets.FontTTF)
	if err != nil {
		r.fontFace = basicfont.Face7x13
		if r.Logger != nil {
			r.Logger.Errorf("fb", "font parse failed, using basicfont: %v", err)
		}
	} else {
		face, ferr := opentype.NewFace(fnt, &opentype.FaceOptions{Size: 48, DPI: 96, Hinting: font.HintingFull})
		if ferr != nil {
			r.fontFace = basicfont.Face7x13
			if r.Logger != nil {
				r.Logger.Errorf("fb", "font face create failed, using basicfont: %v", ferr)
			}
		} else {
			r.fontFace = face
			if r.Logger != nil {
				r.Logger.Infof("fb", "loaded OTF font at 48pt")
			}
		}
	}
	// Also try parsing truetype for freetype renderer
	if tt, terr := truetype.Parse(assets.FontTTF); terr != nil {
		if r.Logger != nil {
			r.Logger.Errorf("fb", "truetype parse failed: %v", terr)
		}
	} else {
		r.ttFont = tt
		if r.Logger != nil {
			r.Logger.Infof("fb", "truetype font parsed for freetype")
		}
	}

	// Decode logo
	if !r.NoLogo {
		img, lerr := png.Decode(bytes.NewReader(assets.LogoPNG))
		if lerr != nil {
			if r.Logger != nil {
				r.Logger.Errorf("fb", "logo load failed: %v", lerr)
			}
		} else {
			r.logo = img
			if r.Logger != nil {
				r.Logger.Infof("fb", "logo loaded")
			}
		}
	}

	r.running.Store(true)
	return nil
}

func (r *FBRenderer) Stop() error {
	r.running.Store(false)
	if r.fbDev != nil {
		r.fbDev.Close()
	}
	return nil
}

// SetScreen sets the current logical screen to be drawn.
func (r *FBRenderer) SetScreen(s Screen) { r.current = s }

// Redraw triggers a draw of the current screen.
func (r *FBRenderer) RedrawWithState(snap state.State) {
	if !r.running.Load() || r.current == nil || r.fbDev == nil {
		return
	}
	// Clear canvas to background each frame for consistent rendering
	r.FillBackground()
	// Provide a Drawer implementation and ask the screen to draw
	r.current.Draw(r, snap)
	_ = blitToFB(r.fbDev, r.canvas)
	if r.Logger != nil {
		r.Logger.Infof("fb", "redraw done, phase=%d", snap.Phase)
	}
}

// RunLoop continuously redraws at ~30 FPS until the context is done.
func (r *FBRenderer) RunLoop(ctx context.Context, store *state.Store) {
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	lastLog := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := store.Snapshot()
			r.RedrawWithState(snap)
			if r.Logger != nil && time.Since(lastLog) > time.Second {
				r.Logger.Infof("fb", "heartbeat frame, phase=%d", snap.Phase)
				lastLog = time.Now()
			}
		}
	}
}

// Drawer primitives
func (r *FBRenderer) FillBackground() {
	draw.Draw(r.canvas, r.canvas.Bounds(), &image.Uniform{C: Background}, image.Point{}, draw.Src)
}

func (r *FBRenderer) DrawLogoCenteredTop() {
	if r.logo == nil {
		return
	}
	// Limit logo to 25% of canvas width and center on screen
	maxW := int(float64(CanvasWidth) * 0.25)
	scale := 1.0
	lw := r.logo.Bounds().Dx()
	lh := r.logo.Bounds().Dy()
	if lw > maxW {
		scale = float64(maxW) / float64(lw)
	}
	sw := int(float64(lw) * scale)
	sh := int(float64(lh) * scale)
	// Center vertically and horizontally
	dst := image.Rect((CanvasWidth-sw)/2, (CanvasHeight-sh)/2-(sh/4), (CanvasWidth-sw)/2+sw, (CanvasHeight-sh)/2-(sh/4)+sh)
	r.lastLogoRect = dst
	// Scale into a temporary RGBA and composite with alpha
	tmp := image.NewRGBA(dst)
	xdraw.NearestNeighbor.Scale(tmp, tmp.Bounds(), r.logo, r.logo.Bounds(), xdraw.Over, nil)
	draw.Draw(r.canvas, dst, tmp, tmp.Bounds().Min, draw.Over)
}

func (r *FBRenderer) DrawTextCentered(text string) {
	// Ensure we have a font face
	if r.fontFace == nil {
		r.fontFace = basicfont.Face7x13
		if r.Logger != nil {
			r.Logger.Errorf("fb", "fontFace nil at draw, defaulting to basicfont")
		}
	}
	// Position text below logo using metrics
	metrics := r.fontFace.Metrics()
	ascent := metrics.Ascent.Ceil()
	margin := 40
	baseline := 0
	// If logoRect is not set or off-screen, use vertical center
	if r.lastLogoRect.Empty() || r.lastLogoRect.Max.Y <= 0 {
		baseline = CanvasHeight/2 + ascent/2
	} else {
		logoBottom := r.lastLogoRect.Max.Y
		baseline = logoBottom + margin + ascent
	}
	// Measure width using font.Drawer to center horizontally
	// Use Foreground color with explicit alpha channel
	textColor := color.RGBA{R: Foreground.R, G: Foreground.G, B: Foreground.B, A: 255}
	d := &font.Drawer{
		Dst:  r.canvas,
		Src:  image.NewUniform(textColor),
		Face: r.fontFace,
	}
	w := d.MeasureString(text).Ceil()
	x := (CanvasWidth - w) / 2

	// Draw text directly onto canvas
	d.Dot = fixed.Point26_6{X: fixed.Int26_6(x << 6), Y: fixed.Int26_6(baseline << 6)}
	d.DrawString(text)
}

// Helper: nearest-neighbor scale of src into dst rectangle on canvas.
func nnScale(dst draw.Image, rect image.Rectangle, src image.Image) {
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	dw := rect.Dx()
	dh := rect.Dy()
	for y := 0; y < dh; y++ {
		sy := src.Bounds().Min.Y + (y*sh)/dh
		for x := 0; x < dw; x++ {
			sx := src.Bounds().Min.X + (x*sw)/dw
			c := src.At(sx, sy)
			dst.Set(rect.Min.X+x, rect.Min.Y+y, c)
		}
	}
}

// Helper: blit canvas to framebuffer via nearest-neighbor scaling.
func blitToFB(dev *fb.Device, canvas *image.RGBA) error {
	if dev == nil {
		return nil
	}
	bounds := dev.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	// For simplicity, write directly using NN sampling from canvas
	for y := 0; y < h; y++ {
		sy := (y * CanvasHeight) / h
		for x := 0; x < w; x++ {
			sx := (x * CanvasWidth) / w
			c := canvas.RGBAAt(sx, sy)
			dev.Set(bounds.Min.X+x, bounds.Min.Y+y, color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xFF})
		}
	}
	return nil
}

// Helper: centered text drawing with foreground color and font face.
func drawTextCentered(img *image.RGBA, text string, baselineY int, fg color.Color, face font.Face) {
	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{C: fg},
		Face: face,
	}
	w := d.MeasureString(text).Ceil()
	x := (CanvasWidth - w) / 2
	d.Dot = fixed.P(x<<6, baselineY<<6)
	d.DrawString(text)
}
func drawTextWithOffset(img *image.RGBA, text string, baselineY int, fg color.Color, face font.Face, offX, offY int) {
	// draw shadow
	shadow := color.RGBA{R: 0, G: 0, B: 0, A: 0xFF}
	drawTextAt(img, text, baselineY+offY, shadow, face, offX)
	// draw main
	drawTextAt(img, text, baselineY, fg, face, 0)
}
func drawTextAt(img *image.RGBA, text string, baselineY int, fg color.Color, face font.Face, xOffset int) {
	d := &font.Drawer{Dst: img, Src: &image.Uniform{C: fg}, Face: face}
	w := d.MeasureString(text).Ceil()
	x := (CanvasWidth - w) / 2
	x += xOffset
	d.Dot = fixed.P(x<<6, baselineY<<6)
	d.DrawString(text)
}
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
