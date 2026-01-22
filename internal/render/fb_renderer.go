package render

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
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
	fbDev     *fb.Device
	canvas    *image.RGBA
	fontFace  font.Face
	otFont    *opentype.Font
	faceCache map[int]font.Face
	ttFont    *truetype.Font
	// Logo is decoded once from embedded assets and can be reused by screens.
	Logo         image.Image
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

func (renderer *FBRenderer) Start(ctx context.Context) error {
	// Open framebuffer
	fbDevice, openErr := fb.Open("/dev/fb0")
	if openErr != nil {
		return openErr
	}
	renderer.fbDev = fbDevice
	if renderer.Logger != nil {
		deviceBounds := fbDevice.Bounds()
		renderer.Logger.Infof("fb", "framebuffer open, bounds=%dx%d", deviceBounds.Dx(), deviceBounds.Dy())
	}

	// Prepare logical canvas
	renderer.canvas = image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))

	// Load font from embedded bytes
	parsedOTFont, parseErr := opentype.Parse(assets.FontTTF)
	if parseErr != nil {
		renderer.fontFace = basicfont.Face7x13
		if renderer.Logger != nil {
			renderer.Logger.Errorf("fb", "font parse failed, using basicfont: %v", parseErr)
		}
	} else {
		renderer.otFont = parsedOTFont
		renderer.fontFace = renderer.faceForSize(48)
		if renderer.Logger != nil {
			renderer.Logger.Infof("fb", "loaded OTF font")
		}
	}
	// Also try parsing truetype for freetype renderer
	if tt, terr := truetype.Parse(assets.FontTTF); terr != nil {
		if renderer.Logger != nil {
			renderer.Logger.Errorf("fb", "truetype parse failed: %v", terr)
		}
	} else {
		renderer.ttFont = tt
		if renderer.Logger != nil {
			renderer.Logger.Infof("fb", "truetype font parsed for freetype")
		}
	}

	// Decode logo
	if !renderer.NoLogo {
		logoImage, logoErr := png.Decode(bytes.NewReader(assets.LogoPNG))
		if logoErr != nil {
			if renderer.Logger != nil {
				renderer.Logger.Errorf("fb", "logo load failed: %v", logoErr)
			}
		} else {
			renderer.Logo = logoImage
			if renderer.Logger != nil {
				renderer.Logger.Infof("fb", "logo loaded")
			}
		}
	}

	renderer.running.Store(true)
	return nil
}

func (renderer *FBRenderer) Stop() error {
	renderer.running.Store(false)
	for _, face := range renderer.faceCache {
		if closer, ok := face.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	renderer.faceCache = nil
	renderer.otFont = nil
	if renderer.fbDev != nil {
		renderer.fbDev.Close()
	}
	return nil
}

// SetScreen sets the current logical screen to be drawn.
func (renderer *FBRenderer) SetScreen(screen Screen) { renderer.current = screen }

func (renderer *FBRenderer) Size() (width int, height int) {
	if renderer.canvas == nil {
		return CanvasWidth, CanvasHeight
	}
	canvasBounds := renderer.canvas.Bounds()
	return canvasBounds.Dx(), canvasBounds.Dy()
}

// Redraw triggers a draw of the current screen.
func (renderer *FBRenderer) RedrawWithState(snap state.State) {
	if !renderer.running.Load() || renderer.current == nil || renderer.fbDev == nil {
		return
	}
	// Clear canvas to background each frame for consistent rendering
	renderer.FillBackground()
	// Provide a Drawer implementation and ask the screen to draw
	renderer.current.Draw(renderer, snap)
	_ = blitToFB(renderer.fbDev, renderer.canvas)
	if renderer.Logger != nil {
		renderer.Logger.Infof("fb", "redraw done, phase=%d", snap.Phase)
	}
}

// RunLoop continuously redraws at ~30 FPS until the context is done.
func (renderer *FBRenderer) RunLoop(ctx context.Context, store *state.Store) {
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	lastLog := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := store.Snapshot()
			renderer.RedrawWithState(snap)
			if renderer.Logger != nil && time.Since(lastLog) > time.Second {
				renderer.Logger.Infof("fb", "heartbeat frame, phase=%d", snap.Phase)
				lastLog = time.Now()
			}
		}
	}
}

// Drawer primitives
func (renderer *FBRenderer) FillBackground() {
	draw.Draw(renderer.canvas, renderer.canvas.Bounds(), &image.Uniform{C: Background}, image.Point{}, draw.Src)
}

func (renderer *FBRenderer) faceForSize(size int) font.Face {
	if size <= 0 {
		size = 48
	}
	if renderer.otFont == nil {
		return basicfont.Face7x13
	}
	if renderer.faceCache == nil {
		renderer.faceCache = map[int]font.Face{}
	}
	if face, ok := renderer.faceCache[size]; ok {
		return face
	}
	fontFace, faceErr := opentype.NewFace(renderer.otFont, &opentype.FaceOptions{Size: float64(size), DPI: 96, Hinting: font.HintingFull})
	if faceErr != nil {
		if renderer.Logger != nil {
			renderer.Logger.Errorf("fb", "font face create failed (size=%d), using basicfont: %v", size, faceErr)
		}
		return basicfont.Face7x13
	}
	renderer.faceCache[size] = fontFace
	return fontFace
}

func (renderer *FBRenderer) MeasureText(text string, style TextStyle) TextMetrics {
	fontFace := renderer.faceForSize(style.Size)
	faceMetrics := fontFace.Metrics()
	ascent := faceMetrics.Ascent.Ceil()
	descent := faceMetrics.Descent.Ceil()
	height := ascent + descent
	lineHeight := faceMetrics.Height.Ceil()

	fontDrawer := &font.Drawer{Face: fontFace}
	width := fontDrawer.MeasureString(text).Ceil()
	return TextMetrics{Width: width, Height: height, Ascent: ascent, Descent: descent, LineHeight: lineHeight}
}

func (renderer *FBRenderer) DrawText(text string, x, y int, style TextStyle) TextMetrics {
	if renderer.canvas == nil {
		return TextMetrics{}
	}
	if style.Color == nil {
		style.Color = Foreground
	}
	fontFace := renderer.faceForSize(style.Size)
	textMetrics := renderer.MeasureText(text, style)

	xPos := x
	switch style.Align {
	case TextAlignCenter:
		xPos = x - (textMetrics.Width / 2)
	case TextAlignRight:
		xPos = x - textMetrics.Width
	default:
		// left
	}
	baseline := y + textMetrics.Ascent

	fontDrawer := &font.Drawer{Dst: renderer.canvas, Src: image.NewUniform(style.Color), Face: fontFace}
	// fixed.P expects pixel coordinates and internally converts to 26.6 fixed-point.
	fontDrawer.Dot = fixed.P(xPos, baseline)
	fontDrawer.DrawString(text)
	return textMetrics
}

func (renderer *FBRenderer) ImageSize(img image.Image) (width int, height int) {
	if img == nil {
		return 0, 0
	}
	imageBounds := img.Bounds()
	return imageBounds.Dx(), imageBounds.Dy()
}

func (renderer *FBRenderer) DrawImage(img image.Image, x, y int, _ ImageOpts) {
	if renderer.canvas == nil || img == nil {
		return
	}
	srcBounds := img.Bounds()
	dstRect := image.Rect(x, y, x+srcBounds.Dx(), y+srcBounds.Dy())
	draw.Draw(renderer.canvas, dstRect, img, srcBounds.Min, draw.Over)
}

func (renderer *FBRenderer) DrawImageInRect(img image.Image, destinationRect image.Rectangle, mode ScaleMode) {
	if renderer.canvas == nil || img == nil {
		return
	}
	if destinationRect.Empty() {
		return
	}
	srcBounds := img.Bounds()
	srcWidth, srcHeight := srcBounds.Dx(), srcBounds.Dy()
	if srcWidth <= 0 || srcHeight <= 0 {
		return
	}

	// Clip destination to canvas.
	destinationRect = destinationRect.Intersect(renderer.canvas.Bounds())
	if destinationRect.Empty() {
		return
	}

	switch mode {
	case ScaleModeStretch:
		xdraw.NearestNeighbor.Scale(renderer.canvas, destinationRect, img, srcBounds, xdraw.Over, nil)
		return
	case ScaleModeFill:
		// Crop source so that scaled output fully covers rect.
		dstWidth, dstHeight := destinationRect.Dx(), destinationRect.Dy()
		scaleX := float64(dstWidth) / float64(srcWidth)
		scaleY := float64(dstHeight) / float64(srcHeight)
		scale := scaleX
		if scaleY > scale {
			scale = scaleY
		}
		cropWidth := int(float64(dstWidth) / scale)
		cropHeight := int(float64(dstHeight) / scale)
		if cropWidth <= 0 || cropHeight <= 0 {
			return
		}
		cropX := srcBounds.Min.X + (srcWidth-cropWidth)/2
		cropY := srcBounds.Min.Y + (srcHeight-cropHeight)/2
		srcCropRect := image.Rect(cropX, cropY, cropX+cropWidth, cropY+cropHeight)
		xdraw.NearestNeighbor.Scale(renderer.canvas, destinationRect, img, srcCropRect, xdraw.Over, nil)
		return
	default:
		// Fit
		dstWidth, dstHeight := destinationRect.Dx(), destinationRect.Dy()
		scaleX := float64(dstWidth) / float64(srcWidth)
		scaleY := float64(dstHeight) / float64(srcHeight)
		scale := scaleX
		if scaleY < scale {
			scale = scaleY
		}
		scaledWidth := int(float64(srcWidth) * scale)
		scaledHeight := int(float64(srcHeight) * scale)
		if scaledWidth <= 0 || scaledHeight <= 0 {
			return
		}
		destX := destinationRect.Min.X + (dstWidth-scaledWidth)/2
		destY := destinationRect.Min.Y + (dstHeight-scaledHeight)/2
		scaledDstRect := image.Rect(destX, destY, destX+scaledWidth, destY+scaledHeight)
		xdraw.NearestNeighbor.Scale(renderer.canvas, scaledDstRect, img, srcBounds, xdraw.Over, nil)
		return
	}
}

func (renderer *FBRenderer) DrawLogoCenteredTop() {
	if renderer.Logo == nil {
		return
	}
	canvasWidth, canvasHeight := renderer.Size()
	// Limit logo to 25% of canvas width.
	maxLogoWidth := int(float64(canvasWidth) * 0.25)
	logoWidth, logoHeight := renderer.Logo.Bounds().Dx(), renderer.Logo.Bounds().Dy()
	if logoWidth <= 0 || logoHeight <= 0 {
		return
	}
	scale := 1.0
	if logoWidth > maxLogoWidth {
		scale = float64(maxLogoWidth) / float64(logoWidth)
	}
	scaledLogoWidth := int(float64(logoWidth) * scale)
	scaledLogoHeight := int(float64(logoHeight) * scale)
	destX := (canvasWidth - scaledLogoWidth) / 2
	destY := (canvasHeight-scaledLogoHeight)/2 - (scaledLogoHeight / 4)
	logoRect := image.Rect(destX, destY, destX+scaledLogoWidth, destY+scaledLogoHeight)
	renderer.lastLogoRect = logoRect
	renderer.DrawImageInRect(renderer.Logo, logoRect, ScaleModeStretch)
}

func (renderer *FBRenderer) DrawTextCentered(text string) {
	canvasWidth, canvasHeight := renderer.Size()
	style := TextStyle{Color: Foreground, Size: 48, Align: TextAlignCenter}
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		textMetrics := renderer.MeasureText(text, style)
		margin := 40
		baseline := 0
		if renderer.lastLogoRect.Empty() || renderer.lastLogoRect.Max.Y <= 0 {
			baseline = canvasHeight/2 + (textMetrics.Ascent / 2)
		} else {
			baseline = renderer.lastLogoRect.Max.Y + margin + textMetrics.Ascent
		}
		textTopY := baseline - textMetrics.Ascent
		renderer.DrawText(text, canvasWidth/2, textTopY, style)
		return
	}

	// Multiline: stack each line below the previous one.
	// Use a consistent line height based on the current style.
	lineMetrics := renderer.MeasureText("Ag", style)
	lineHeight := lineMetrics.LineHeight
	if lineHeight <= 0 {
		lineHeight = lineMetrics.Height
	}
	if lineHeight <= 0 {
		lineHeight = 1
	}
	blockHeight := lineHeight * len(lines)
	margin := 40
	startY := 0
	if renderer.lastLogoRect.Empty() || renderer.lastLogoRect.Max.Y <= 0 {
		startY = (canvasHeight / 2) - (blockHeight / 2)
	} else {
		startY = renderer.lastLogoRect.Max.Y + margin
	}

	for i, line := range lines {
		renderer.DrawText(line, canvasWidth/2, startY+i*lineHeight, style)
	}
}

// Helper: nearest-neighbor scale of src into dst rectangle on canvas.
func nnScale(dst draw.Image, rect image.Rectangle, src image.Image) {
	srcWidth := src.Bounds().Dx()
	srcHeight := src.Bounds().Dy()
	dstWidth := rect.Dx()
	dstHeight := rect.Dy()
	for y := 0; y < dstHeight; y++ {
		sy := src.Bounds().Min.Y + (y*srcHeight)/dstHeight
		for x := 0; x < dstWidth; x++ {
			sx := src.Bounds().Min.X + (x*srcWidth)/dstWidth
			pixelColor := src.At(sx, sy)
			dst.Set(rect.Min.X+x, rect.Min.Y+y, pixelColor)
		}
	}
}

// Helper: blit canvas to framebuffer via nearest-neighbor scaling.
func blitToFB(dev *fb.Device, canvas *image.RGBA) error {
	if dev == nil {
		return nil
	}
	bounds := dev.Bounds()
	fbWidth := bounds.Dx()
	fbHeight := bounds.Dy()
	// For simplicity, write directly using NN sampling from canvas
	for y := 0; y < fbHeight; y++ {
		sy := (y * CanvasHeight) / fbHeight
		for x := 0; x < fbWidth; x++ {
			sx := (x * CanvasWidth) / fbWidth
			pixel := canvas.RGBAAt(sx, sy)
			dev.Set(bounds.Min.X+x, bounds.Min.Y+y, color.RGBA{R: pixel.R, G: pixel.G, B: pixel.B, A: 0xFF})
		}
	}
	return nil
}

// Helper: centered text drawing with foreground color and font face.
func drawTextCentered(img *image.RGBA, text string, baselineY int, fg color.Color, face font.Face) {
	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{C: fg},
		Face: face,
	}
	textWidth := drawer.MeasureString(text).Ceil()
	xPos := (CanvasWidth - textWidth) / 2
	drawer.Dot = fixed.P(xPos, baselineY)
	drawer.DrawString(text)
}
func drawTextWithOffset(img *image.RGBA, text string, baselineY int, fg color.Color, face font.Face, offX, offY int) {
	// draw shadow
	shadow := color.RGBA{R: 0, G: 0, B: 0, A: 0xFF}
	drawTextAt(img, text, baselineY+offY, shadow, face, offX)
	// draw main
	drawTextAt(img, text, baselineY, fg, face, 0)
}
func drawTextAt(img *image.RGBA, text string, baselineY int, fg color.Color, face font.Face, xOffset int) {
	drawer := &font.Drawer{Dst: img, Src: &image.Uniform{C: fg}, Face: face}
	textWidth := drawer.MeasureString(text).Ceil()
	xPos := (CanvasWidth - textWidth) / 2
	xPos += xOffset
	drawer.Dot = fixed.P(xPos, baselineY)
	drawer.DrawString(text)
}
func bytesReader(data []byte) *bytes.Reader { return bytes.NewReader(data) }
