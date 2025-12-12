package render

import "image/color"

// Global render configuration for colors and logical canvas.
var (
    // Foreground and background per user-provided hex values.
    Foreground = color.RGBA{R: 0x90, G: 0x00, B: 0xFF, A: 0xFF} // #9000ff
    Background = color.RGBA{R: 0xFF, G: 0xDC, B: 0x00, A: 0xFF} // #ffdc00

    // Logical canvas size; scaled to framebuffer.
    CanvasWidth  = 1920
    CanvasHeight = 1080
)
