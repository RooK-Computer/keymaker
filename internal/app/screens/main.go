package screens

import (
	"context"
	"image"
	"strings"

	"github.com/rook-computer/keymaker/internal/render"
	"github.com/rook-computer/keymaker/internal/render/layout"
	"github.com/rook-computer/keymaker/internal/state"
)

type MainScreen struct {
	lastHotspotQRPayload string
	hotspotQRImage       image.Image

	lastURLQRPayload string
	urlQRImage       image.Image
}

func (screen *MainScreen) Start(ctx context.Context) error {
	return nil
}

func (screen *MainScreen) Stop() error { return nil }

func (screen *MainScreen) Draw(drawer render.Drawer, currentState state.State) {
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
		if payload != "" && payload != screen.lastHotspotQRPayload {
			qrImage, err := render.GenerateQRCodeImage(payload, 512)
			if err == nil {
				screen.hotspotQRImage = qrImage
				screen.lastHotspotQRPayload = payload
			}
		}

		if screen.hotspotQRImage != nil {
			// Use remaining space below text for the QR code.
			qrRect := image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y)
			qrRect = layout.Inset(qrRect, 8)
			qrRect = layout.FitSquare(qrRect)
			drawer.DrawImageInRect(screen.hotspotQRImage, qrRect, render.ScaleModeFit)
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
	if payload != "" && payload != screen.lastURLQRPayload {
		qrImage, err := render.GenerateQRCodeImage(payload, 512)
		if err == nil {
			screen.urlQRImage = qrImage
			screen.lastURLQRPayload = payload
		}
	}

	if screen.urlQRImage == nil {
		return
	}

	qrRect := image.Rect(rect.Min.X, y, rect.Max.X, rect.Max.Y)
	qrRect = layout.Inset(qrRect, 8)
	qrRect = layout.FitSquare(qrRect)
	drawer.DrawImageInRect(screen.urlQRImage, qrRect, render.ScaleModeFit)
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
