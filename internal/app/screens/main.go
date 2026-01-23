package screens

import (
	"context"
	"image"
	"strings"
	"sync"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/render/layout"
	"github.com/rook-computer/keymaker/internal/state"
)

type MainScreen struct {
	lastHotspotQRPayload      string
	hotspotQRImage            image.Image
	requestedHotspotQRPayload string

	lastURLQRPayload      string
	urlQRImage            image.Image
	requestedURLQRPayload string

	qrReqCh chan qrRequest
	qrResCh chan qrResult
	qrStop  context.CancelFunc

	mu sync.Mutex
}

type qrRequest struct {
	kind    string
	payload string
	size    int
}

type qrResult struct {
	kind    string
	payload string
	img     image.Image
	err     error
}

func (screen *MainScreen) Start(ctx context.Context) error {
	screen.mu.Lock()
	defer screen.mu.Unlock()

	// Lazily start a single worker for this screen instance.
	if screen.qrReqCh != nil {
		return nil
	}

	workerCtx, cancel := context.WithCancel(ctx)
	screen.qrStop = cancel
	screen.qrReqCh = make(chan qrRequest, 8)
	screen.qrResCh = make(chan qrResult, 8)

	go func() {
		for {
			select {
			case <-workerCtx.Done():
				return
			case req := <-screen.qrReqCh:
				img, err := render.GenerateQRCodeImage(req.payload, req.size)
				select {
				case screen.qrResCh <- qrResult{kind: req.kind, payload: req.payload, img: img, err: err}:
				case <-workerCtx.Done():
					return
				}
			}
		}
	}()
	return nil
}

func (screen *MainScreen) Stop() error {
	screen.mu.Lock()
	stop := screen.qrStop
	screen.qrStop = nil
	screen.qrReqCh = nil
	screen.qrResCh = nil
	screen.requestedHotspotQRPayload = ""
	screen.requestedURLQRPayload = ""
	screen.mu.Unlock()

	if stop != nil {
		stop()
	}
	return nil
}

func (screen *MainScreen) drainQRResults() {
	screen.mu.Lock()
	resCh := screen.qrResCh
	requestedHotspot := screen.requestedHotspotQRPayload
	requestedURL := screen.requestedURLQRPayload
	screen.mu.Unlock()

	if resCh == nil {
		return
	}

	for {
		select {
		case res := <-resCh:
			if res.err != nil {
				// Ignore; keep previous image if any.
				continue
			}

			screen.mu.Lock()
			switch res.kind {
			case "hotspot":
				if res.payload == requestedHotspot {
					screen.hotspotQRImage = res.img
					screen.lastHotspotQRPayload = res.payload
					screen.requestedHotspotQRPayload = ""
				}
			case "url":
				if res.payload == requestedURL {
					screen.urlQRImage = res.img
					screen.lastURLQRPayload = res.payload
					screen.requestedURLQRPayload = ""
				}
			}
			screen.mu.Unlock()
		default:
			return
		}
	}
}

func (screen *MainScreen) requestQR(kind, payload string, size int) {
	screen.mu.Lock()
	reqCh := screen.qrReqCh
	screen.mu.Unlock()

	if reqCh == nil {
		return
	}

	select {
	case reqCh <- qrRequest{kind: kind, payload: payload, size: size}:
	default:
		// If queue is full, skip this frame; we'll retry next draw.
	}
}

func (screen *MainScreen) Draw(drawer render.Drawer, currentState state.State) {
	// Apply any completed QR results without blocking rendering.
	screen.drainQRResults()

	drawer.FillBackground()

	canvasWidth, canvasHeight := drawer.Size()
	root := image.Rect(0, 0, canvasWidth, canvasHeight)
	content := layout.Inset(root, 48)
	quads := layout.Grid2x2(content)

	// Top-left: logo.
	var logo image.Image
	if fbRenderer, ok := drawer.(*render.FBRenderer); ok {
		logo = fbRenderer.Logo
	}
	if logo != nil {
		// Give the logo a reasonable maximum footprint within the quadrant.
		logoSlot := layout.Inset(quads.TopLeft, 16)
		maxLogoWidth := int(float64(logoSlot.Dx()) * 0.85)
		maxLogoHeight := int(float64(logoSlot.Dy()) * 0.85)
		logoRect := layout.AnchorTopLeft(logoSlot, maxLogoWidth, maxLogoHeight)
		drawer.DrawImageInRect(logo, logoRect, render.ScaleModeFit)
	}

	// Top-right: WiFi info.
	screen.drawWiFi(drawer, layout.Inset(quads.TopRight, 16), currentState)

	// Bottom-left: IP info.
	screen.drawIP(drawer, layout.Inset(quads.BottomLeft, 16), currentState)

	// Bottom-right: cartridge info.
	screen.drawCartridge(drawer, layout.Inset(quads.BottomRight, 16))
}

func (screen *MainScreen) drawWiFi(drawer render.Drawer, rect image.Rectangle, currentState state.State) {
	if rect.Empty() {
		return
	}

	wifiConfigSnapshot := state.GetWiFiConfig().Snapshot()
	hotspotActive := wifiConfigSnapshot.Initialized && wifiConfigSnapshot.Mode == state.WiFiModeHotspot

	headerStyle := render.TextStyle{Size: 36, Align: render.TextAlignLeft}
	bodyStyle := render.TextStyle{Size: 28, Align: render.TextAlignLeft}

	y := rect.Min.Y
	headerMetrics := drawer.DrawText("WIFI", rect.Min.X, y, headerStyle)
	y += headerMetrics.LineHeight + 10

	ssid := strings.TrimSpace(currentState.WiFi.SSID)
	if hotspotActive {
		if ssid == "" {
			drawer.DrawText("Hotspot active", rect.Min.X, y, bodyStyle)
		} else {
			drawer.DrawText("Hotspot: "+ssid, rect.Min.X, y, bodyStyle)
		}
		y += drawer.MeasureText("Hotspot: ", bodyStyle).LineHeight + 10

		payload := buildOpenWiFiQRPayload(ssid)
		if payload == "" {
			screen.mu.Lock()
			screen.hotspotQRImage = nil
			screen.lastHotspotQRPayload = ""
			screen.requestedHotspotQRPayload = ""
			screen.mu.Unlock()
			return
		}

		screen.mu.Lock()
		needRequest := payload != screen.lastHotspotQRPayload && payload != screen.requestedHotspotQRPayload
		if needRequest {
			screen.requestedHotspotQRPayload = payload
		}
		hasImage := screen.hotspotQRImage != nil && screen.lastHotspotQRPayload == payload
		screen.mu.Unlock()

		if needRequest {
			screen.requestQR("hotspot", payload, 512)
		}

		if hasImage {
			// Use remaining space below text for the QR code.
			qrRect := image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y)
			qrRect = layout.Inset(qrRect, 8)
			qrRect = layout.FitSquare(qrRect)
			screen.mu.Lock()
			img := screen.hotspotQRImage
			screen.mu.Unlock()
			drawer.DrawImageInRect(img, qrRect, render.ScaleModeFit)
		} else {
			drawer.DrawText("Generating QR…", rect.Min.X, y, bodyStyle)
		}
		return
	}

	if ssid == "" {
		drawer.DrawText("Not connected", rect.Min.X, y, bodyStyle)
		return
	}
	drawer.DrawText("Connected: "+ssid, rect.Min.X, y, bodyStyle)
}

func (screen *MainScreen) drawIP(drawer render.Drawer, rect image.Rectangle, currentState state.State) {
	if rect.Empty() {
		return
	}

	headerStyle := render.TextStyle{Size: 36, Align: render.TextAlignLeft}
	bodyStyle := render.TextStyle{Size: 28, Align: render.TextAlignLeft}

	y := rect.Min.Y
	headerMetrics := drawer.DrawText("IP", rect.Min.X, y, headerStyle)
	y += headerMetrics.LineHeight + 10

	ip := strings.TrimSpace(currentState.Network.IP)
	url := strings.TrimSpace(currentState.Network.URL)
	if url == "" && ip != "" {
		url = "http://" + ip
	}

	if ip == "" {
		drawer.DrawText("No network", rect.Min.X, y, bodyStyle)
		return
	}

	drawer.DrawText("IP: "+ip, rect.Min.X, y, bodyStyle)
	y += drawer.MeasureText("IP: ", bodyStyle).LineHeight + 8

	if url != "" {
		drawer.DrawText(url, rect.Min.X, y, bodyStyle)
		y += drawer.MeasureText(url, bodyStyle).LineHeight + 10
	}

	payload := strings.TrimSpace(currentState.Network.URLQR)
	if payload == "" {
		payload = url
	}
	if payload == "" {
		screen.mu.Lock()
		screen.urlQRImage = nil
		screen.lastURLQRPayload = ""
		screen.requestedURLQRPayload = ""
		screen.mu.Unlock()
		return
	}

	screen.mu.Lock()
	needRequest := payload != screen.lastURLQRPayload && payload != screen.requestedURLQRPayload
	if needRequest {
		screen.requestedURLQRPayload = payload
	}
	hasImage := screen.urlQRImage != nil && screen.lastURLQRPayload == payload
	screen.mu.Unlock()

	if needRequest {
		screen.requestQR("url", payload, 512)
	}

	if !hasImage {
		drawer.DrawText("Generating QR…", rect.Min.X, y, bodyStyle)
		return
	}

	qrRect := image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y)
	qrRect = layout.Inset(qrRect, 8)
	qrRect = layout.FitSquare(qrRect)
	screen.mu.Lock()
	img := screen.urlQRImage
	screen.mu.Unlock()
	drawer.DrawImageInRect(img, qrRect, render.ScaleModeFit)
}

func (screen *MainScreen) drawCartridge(drawer render.Drawer, rect image.Rectangle) {
	if rect.Empty() {
		return
	}

	snapshot := state.GetCartridgeInfo().Snapshot()

	headerStyle := render.TextStyle{Size: 36, Align: render.TextAlignLeft}
	bodyStyle := render.TextStyle{Size: 28, Align: render.TextAlignLeft}

	y := rect.Min.Y
	headerMetrics := drawer.DrawText("CARTRIDGE", rect.Min.X, y, headerStyle)
	y += headerMetrics.LineHeight + 10

	if snapshot.Busy {
		drawer.DrawText("Busy", rect.Min.X, y, bodyStyle)
		y += drawer.MeasureText("Busy", bodyStyle).LineHeight + 6
	}

	if !snapshot.Present {
		drawer.DrawText("No cartridge", rect.Min.X, y, bodyStyle)
		return
	}

	mountedText := "Mounted: no"
	if snapshot.Mounted {
		mountedText = "Mounted: yes"
	}
	drawer.DrawText(mountedText, rect.Min.X, y, bodyStyle)
	y += drawer.MeasureText(mountedText, bodyStyle).LineHeight + 6

	retropieText := "RetroPie: no"
	if snapshot.IsRetroPie {
		retropieText = "RetroPie: yes"
	}
	drawer.DrawText(retropieText, rect.Min.X, y, bodyStyle)
}

func buildOpenWiFiQRPayload(ssid string) string {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return ""
	}
	// Standard WiFi QR payload format.
	// Open networks use T:nopass and an empty password.
	return "WIFI:T:nopass;S:" + escapeWiFiQRField(ssid) + ";P:;;"
}

func escapeWiFiQRField(value string) string {
	// According to common WiFi QR conventions, these characters should be escaped.
	value = strings.ReplaceAll(value, `\\`, `\\\\`)
	value = strings.ReplaceAll(value, `;`, `\\;`)
	value = strings.ReplaceAll(value, `,`, `\\,`)
	value = strings.ReplaceAll(value, `:`, `\\:`)
	return value
}
