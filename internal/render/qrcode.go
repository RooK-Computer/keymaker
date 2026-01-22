package render

import (
	"image"

	"github.com/skip2/go-qrcode"
)

const defaultQRCodeSizePx = 256

// GenerateQRCodeImage returns a QR code image for the given payload.
// If payload is empty, it returns (nil, nil).
func GenerateQRCodeImage(payload string, sizePx int) (image.Image, error) {
	if payload == "" {
		return nil, nil
	}
	if sizePx <= 0 {
		sizePx = defaultQRCodeSizePx
	}

	qrCode, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	return qrCode.Image(sizePx), nil
}
