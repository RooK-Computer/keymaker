package render

import (
    "bytes"
    "context"
    "image"
    "image/color"
    "image/draw"
    "image/png"
    "sync/atomic"

    fb "github.com/gonutz/framebuffer"
    "github.com/rook-computer/keymaker/internal/assets"
    "github.com/rook-computer/keymaker/internal/state"
    "golang.org/x/image/font"
    "golang.org/x/image/font/opentype"
    "golang.org/x/image/math/fixed"
    xdraw "golang.org/x/image/draw"
)

// FBRenderer renders to the Linux framebuffer using an offscreen logical canvas.
type FBRenderer struct {
    fbDev   *fb.Device
    canvas  *image.RGBA
    fontFace font.Face
    logo     image.Image
    running  atomic.Bool
    current  Screen
}

func NewFBRenderer() *FBRenderer { return &FBRenderer{} }

func (r *FBRenderer) Start(ctx context.Context) error {
    // Open framebuffer
    dev, err := fb.Open("/dev/fb0")
    if err != nil { return err }
    r.fbDev = dev

    // Prepare logical canvas
    r.canvas = image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))

    // Load font from embedded bytes
    fnt, err := opentype.Parse(assets.FontTTF)
    if err != nil { return err }
    face, err := opentype.NewFace(fnt, &opentype.FaceOptions{Size: 18, DPI: 96, Hinting: font.HintingFull})
    if err != nil { return err }
    r.fontFace = face

    // Decode logo
    img, err := png.Decode(bytes.NewReader(assets.LogoPNG))
    if err != nil { return err }
    r.logo = img

    r.running.Store(true)
    return nil
}

func (r *FBRenderer) Stop() error {
    r.running.Store(false)
    if r.fbDev != nil { r.fbDev.Close() }
    return nil
}

// SetScreen sets the current logical screen to be drawn.
func (r *FBRenderer) SetScreen(s Screen) { r.current = s }

// Redraw triggers a draw of the current screen.
func (r *FBRenderer) Redraw() {
    if r.current == nil { return }
    // Provide a Drawer implementation and ask the screen to draw
    r.current.Draw(r, state.State{})
    _ = blitToFB(r.fbDev, r.canvas)
}

// Drawer primitives
func (r *FBRenderer) FillBackground() {
    draw.Draw(r.canvas, r.canvas.Bounds(), &image.Uniform{C: Background}, image.Point{}, draw.Src)
}

func (r *FBRenderer) DrawLogoCenteredTop() {
    if r.logo == nil { return }
    maxW := int(float64(CanvasWidth) * 0.7)
    scale := 1.0
    lw := r.logo.Bounds().Dx()
    lh := r.logo.Bounds().Dy()
    if lw > maxW { scale = float64(maxW) / float64(lw) }
    sw := int(float64(lw) * scale)
    sh := int(float64(lh) * scale)
    dst := image.Rect((CanvasWidth-sw)/2, CanvasHeight/6, (CanvasWidth-sw)/2+sw, CanvasHeight/6+sh)
    // Scale into a temporary RGBA and composite with alpha
    tmp := image.NewRGBA(dst)
    xdraw.NearestNeighbor.Scale(tmp, tmp.Bounds(), r.logo, r.logo.Bounds(), xdraw.Over, nil)
    draw.Draw(r.canvas, dst, tmp, tmp.Bounds().Min, draw.Over)
}

func (r *FBRenderer) DrawTextCentered(text string) {
    drawTextCentered(r.canvas, text, CanvasHeight/6+80, Foreground, r.fontFace)
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
    if dev == nil { return nil }
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
        Dst: img,
        Src: &image.Uniform{C: fg},
        Face: face,
    }
    w := d.MeasureString(text).Ceil()
    x := (CanvasWidth - w) / 2
    d.Dot = fixed.P(x<<6, baselineY<<6)
    d.DrawString(text)
}
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
