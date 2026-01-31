package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rook-computer/keymaker/internal/assets"
)

type HTTPServer struct {
	Addr string

	// DevMode enables development conveniences such as CORS and extra logging.
	DevMode bool

	// StaticDir, when set to an existing directory, is served at "/".
	// The API remains available under /api/v1/.
	StaticDir string

	// Handler, when set, is used as-is for the HTTP server.
	// This lets binaries register routes without coupling it to HTTPServer startup.
	Handler http.Handler

	// EjectFunc is called by the API when POST /api/v1/eject is invoked.
	// It should switch the UI to the eject screen and prepare the cartridge.
	EjectFunc func(ctx context.Context) error

	// FlashFunc is called by the API when POST /api/v1/flash is invoked.
	// The body is expected to be a gzipped disk image and must be streamed.
	FlashFunc func(ctx context.Context, reader io.Reader) error

	mu     sync.Mutex
	srv    *http.Server
	ln     net.Listener
	closed bool
}

func NewHTTPServer(cfg ServerConfig) *HTTPServer {
	return &HTTPServer{Addr: cfg.ListenAddr, DevMode: cfg.DevMode}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("web server already stopped")
	}
	if s.srv != nil {
		return nil
	}

	addr := s.Addr
	if addr == "" {
		addr = ":80"
	}

	handler := s.Handler
	if handler == nil {
		handler = NewDefaultMux(s.StaticDir, APIV1Config{Handlers: APIV1Handlers{EjectFunc: s.EjectFunc, FlashFunc: s.FlashFunc}, Deps: NewDeviceAPIV1Deps(nil)})
	}
	if s.DevMode {
		handler = WithDevCORS(handler)
	}

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.srv = nil
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	s.ln = ln

	go func() {
		<-ctx.Done()
		_ = s.Stop()
	}()

	go func() {
		err := s.srv.Serve(ln)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return
		}
		// No logger plumbed into web package yet; ignore here.
	}()

	return nil
}

func (s *HTTPServer) Stop() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	srv := s.srv
	ln := s.ln
	s.srv = nil
	s.ln = nil
	s.mu.Unlock()

	if ln != nil {
		_ = ln.Close()
	}
	if srv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func (s *HTTPServer) staticHandler() http.Handler {
	return StaticUIHandler(s.StaticDir)
}

func StaticUIHandler(staticDir string) http.Handler {
	if staticDir == "" {
		fileServer := http.FileServer(http.FS(assets.WebUI))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Clean path to avoid oddities.
			r.URL.Path = filepath.ToSlash(filepath.Clean("/" + r.URL.Path))
			fileServer.ServeHTTP(w, r)
		})
	}

	// When StaticDir is set to an existing directory, serve it at '/'.
	if st, err := os.Stat(staticDir); err != nil || !st.IsDir() {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	fs := http.Dir(staticDir)
	fileServer := http.FileServer(fs)

	// When serving at '/', ensure we don't accidentally expose parent directory traversal.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean path to avoid oddities.
		r.URL.Path = filepath.ToSlash(filepath.Clean("/" + r.URL.Path))
		fileServer.ServeHTTP(w, r)
	})
}
