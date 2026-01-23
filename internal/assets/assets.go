package assets

import (
	"embed"
	"io/fs"
)

//go:embed logo.png
var LogoPNG []byte

//go:embed BerkeleyMonoTrial-Regular.otf
var FontTTF []byte

//go:embed web
var webFS embed.FS

// WebUI is an embedded filesystem rooted at internal/assets/web.
// It contains index.html and the Vite build output under assets/.
var WebUI fs.FS

func init() {
	// Embed paths include the leading directory; strip it for serving at '/'.
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	WebUI = sub
}
