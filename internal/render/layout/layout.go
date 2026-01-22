package layout

import "image"

// Inset shrinks rect by paddingPx on all sides.
func Inset(rect image.Rectangle, paddingPx int) image.Rectangle {
	if paddingPx <= 0 {
		return rect
	}
	out := image.Rect(rect.Min.X+paddingPx, rect.Min.Y+paddingPx, rect.Max.X-paddingPx, rect.Max.Y-paddingPx)
	return Normalize(out)
}

// Normalize ensures Min is <= Max on both axes.
func Normalize(rect image.Rectangle) image.Rectangle {
	if rect.Min.X > rect.Max.X {
		rect.Min.X, rect.Max.X = rect.Max.X, rect.Min.X
	}
	if rect.Min.Y > rect.Max.Y {
		rect.Min.Y, rect.Max.Y = rect.Max.Y, rect.Min.Y
	}
	return rect
}

// SplitVertical splits rect into left and right parts.
// leftWidthPx is clamped to [0, rect.Dx()].
func SplitVertical(rect image.Rectangle, leftWidthPx int) (left image.Rectangle, right image.Rectangle) {
	rect = Normalize(rect)
	width := rect.Dx()
	if leftWidthPx < 0 {
		leftWidthPx = 0
	}
	if leftWidthPx > width {
		leftWidthPx = width
	}
	left = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+leftWidthPx, rect.Max.Y)
	right = image.Rect(rect.Min.X+leftWidthPx, rect.Min.Y, rect.Max.X, rect.Max.Y)
	return left, right
}

// SplitHorizontal splits rect into top and bottom parts.
// topHeightPx is clamped to [0, rect.Dy()].
func SplitHorizontal(rect image.Rectangle, topHeightPx int) (top image.Rectangle, bottom image.Rectangle) {
	rect = Normalize(rect)
	height := rect.Dy()
	if topHeightPx < 0 {
		topHeightPx = 0
	}
	if topHeightPx > height {
		topHeightPx = height
	}
	top = image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+topHeightPx)
	bottom = image.Rect(rect.Min.X, rect.Min.Y+topHeightPx, rect.Max.X, rect.Max.Y)
	return top, bottom
}

type Grid2x2Rects struct {
	TopLeft     image.Rectangle
	TopRight    image.Rectangle
	BottomLeft  image.Rectangle
	BottomRight image.Rectangle
}

// Grid2x2 splits rect into four equal quadrants.
func Grid2x2(rect image.Rectangle) Grid2x2Rects {
	rect = Normalize(rect)
	midX := rect.Min.X + rect.Dx()/2
	midY := rect.Min.Y + rect.Dy()/2
	return Grid2x2Rects{
		TopLeft:     image.Rect(rect.Min.X, rect.Min.Y, midX, midY),
		TopRight:    image.Rect(midX, rect.Min.Y, rect.Max.X, midY),
		BottomLeft:  image.Rect(rect.Min.X, midY, midX, rect.Max.Y),
		BottomRight: image.Rect(midX, midY, rect.Max.X, rect.Max.Y),
	}
}

// AnchorTopLeft returns a rectangle of size (widthPx,heightPx) placed in the top-left of rect.
func AnchorTopLeft(rect image.Rectangle, widthPx, heightPx int) image.Rectangle {
	rect = Normalize(rect)
	if widthPx < 0 {
		widthPx = 0
	}
	if heightPx < 0 {
		heightPx = 0
	}
	maxW := rect.Dx()
	maxH := rect.Dy()
	if widthPx > maxW {
		widthPx = maxW
	}
	if heightPx > maxH {
		heightPx = maxH
	}
	return image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+widthPx, rect.Min.Y+heightPx)
}

// FitSquare returns the largest square that fits into rect, anchored at the top-left.
func FitSquare(rect image.Rectangle) image.Rectangle {
	rect = Normalize(rect)
	size := rect.Dx()
	if rect.Dy() < size {
		size = rect.Dy()
	}
	if size < 0 {
		size = 0
	}
	return AnchorTopLeft(rect, size, size)
}
