# Rook Cartridge Writer Assistant, Implementation Plan 5: better rendering

The software we're going to build here will be run on a custom retro gaming console which utilizes cartridges.
It is built on Raspberry PI CM4 Modules and the cartridges are essentially SD Card slots. 
It will be put into a cartridge which copies itself into RAM. Afterwards the user can install a cartridge he wants to change the contents of.
More details will be added lateron.

This implementation plan has a heritage; all prior ones are considered common knowledge.

We are still in the phase of building the foundation. As this is running on a console without X and without classic GUI libraries, the renderer needs to step up a bit.

At the moment, screens only have a very limited set of primitives (mostly centered logo + centered text). That is fine for the first screens, but it will not scale to the upcoming ones (WiFi credentials, IP address, QR codes, button hints, progress, error details, etc.).

## Goals

* Provide a small but flexible set of drawing primitives to screens.
* Keep screens independent from framebuffer details (still draw via a `render.Drawer`).
* Make it trivial to place text and images at explicit coordinates.
* Keep the current convenience helpers (centered logo/text), but implement them using the generic primitives.

## Non-goals (for this step)

* A full UI toolkit (layout engine, complex widgets, animations).
* Hardware accelerated rendering.
* Perfect typography (kerning tuning, shaping, multi-font support).

## Design decisions

* The logical coordinate system is the existing canvas (see `internal/render/config.go`): origin is the top-left corner, units are pixels.
* Screen code draws in logical coordinates; the renderer keeps scaling to the physical framebuffer.
* All new primitives should be implementable on the offscreen canvas (software draw) and must not expose `/dev/fb0` or the framebuffer library to screens.

## New primitives to add

### 1) Rendering area / canvas metrics

Add a way for a screen to query the drawable area.

Suggested API additions to the Drawer:

* `Size() (width int, height int)`
	* Used by screens to place content without hardcoding `CanvasWidth`/`CanvasHeight`.

### 2) Generic text rendering

We need to be able to draw text at given locations.

Add (at minimum) these capabilities:

* Draw text at an explicit position.
* Measure text width (and preferably height/line height).

Suggested API (shape can differ, but it should support these features):

* `MeasureText(text string, style TextStyle) TextMetrics`
* `DrawText(text string, x, y int, style TextStyle) TextMetrics`

Where:

* `TextStyle` should include at least:
	* font size (or a small enum of sizes)
	* color
	* alignment (left/center/right)
	* anchor (top-left vs baseline-left; pick one and be consistent)
* `TextMetrics` should include at least:
	* width in pixels
	* ascent/descent or total height

Keep a convenience helper:

* `DrawTextCentered(text string)`

but re-implement it via `MeasureText` + `DrawText`.

### 3) Generic image rendering

We need to draw embedded graphics at any given location.

Add (at minimum) these capabilities:

* Fetch image size (width/height).
* Draw an image at a given location.
* Draw an image into a rectangle (scaling to fit).

Suggested API:

* `ImageSize(img ImageRef) (w int, h int)`
* `DrawImage(img ImageRef, x, y int, opts ImageOpts)`
* `DrawImageInRect(img ImageRef, rect Rect, mode ScaleMode)`

Notes:

* `ImageRef` can be either `image.Image` or an internal asset identifier (preferred long-term so screens don’t need to decode PNGs themselves).
* `ScaleMode` should support at least:
	* `Fit` (keep aspect ratio, letterbox)
	* `Fill` (keep aspect ratio, crop)
	* `Stretch` (ignore aspect ratio)

Keep a convenience helper:

* `DrawLogoCenteredTop()`

but re-implement it via the generic image APIs.

## Refactor existing code to use the new primitives

Once the generic primitives exist:

* Update the current framebuffer renderer implementation to implement them.
* Keep `FillBackground()` as-is.
* Rewrite the existing specialized helpers (`DrawLogoCenteredTop`, `DrawTextCentered`) to call the generic primitives.
* Remove or stop using any private “center-only” text helpers that become redundant.

## Acceptance criteria

This plan is complete when:

* A screen can draw multiple independent text blocks (e.g. title + subtitle + footer) at explicit x/y coordinates.
* A screen can draw an embedded image at an explicit x/y coordinate and can scale it into a given rectangle.
* A screen can measure text and image dimensions without hardcoding constants.
* Existing screens still compile and render correctly (centered logo + centered text still works), but their helpers now build on the generic primitives.

## Suggested manual validation

* Temporarily update one of the simple screens to render:
	* a top-left label (e.g. "debug")
	* a centered title
	* a bottom-right footer with smaller font
	* the logo placed in a non-centered location to prove positioning works
* Verify on real hardware that scaling to the framebuffer still looks correct.

Flesh out more generic functions for rendering text and images. Add functions where you see fit (e.g. calculating the width of text, fetching the size of images, returning the size of the rendering area). Once those are done, update the present highly specialized functions to use the generic ones.
