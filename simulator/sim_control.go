package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/web"
)

type SimFaults struct {
	MountFail           bool  `json:"mountFail"`
	EjectFail           bool  `json:"ejectFail"`
	FlashFailAfterBytes int64 `json:"flashFailAfterBytes"`
}

type SimControl struct {
	processCtx      context.Context
	root            string
	startupScenario string
	currentScenario atomic.Value // string

	info   *state.CartridgeInfo
	faults struct {
		mu sync.RWMutex
		v  SimFaults
	}

	reinsertSeq int64
}

func NewSimControl(processCtx context.Context, root, startupScenario string, info *state.CartridgeInfo) *SimControl {
	if processCtx == nil {
		processCtx = context.Background()
	}
	if info == nil {
		info = state.GetCartridgeInfo()
	}
	c := &SimControl{processCtx: processCtx, root: filepath.Clean(root), startupScenario: strings.TrimSpace(startupScenario), info: info}
	if c.startupScenario == "" {
		c.startupScenario = "retropie"
	}
	c.currentScenario.Store(c.startupScenario)
	return c
}

func (c *SimControl) Deps() web.APIV1Deps {
	return web.APIV1Deps{
		Cartridge: c.info,
		Mounter:   SimCartridgeMounter{Control: c},
		RetroPie:  web.FileSystemRetroPieStorage{RomsRoot: filepath.Join(c.root, "home/pi/RetroPie/roms")},
	}
}

func (c *SimControl) ApplyScenario(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = c.startupScenario
	}
	c.info.Reset()
	if err := applyScenario(c.processCtx, c.root, name, c.info); err != nil {
		return err
	}
	c.currentScenario.Store(name)
	return nil
}

func (c *SimControl) Reset() error {
	c.SetFaults(SimFaults{})
	return c.ApplyScenario(c.startupScenario)
}

func (c *SimControl) Faults() SimFaults {
	c.faults.mu.RLock()
	defer c.faults.mu.RUnlock()
	return c.faults.v
}

func (c *SimControl) SetFaults(v SimFaults) {
	c.faults.mu.Lock()
	c.faults.v = v
	c.faults.mu.Unlock()
}

func (c *SimControl) Eject(reqCtx context.Context) error {
	_ = reqCtx
	faults := c.Faults()
	if faults.EjectFail {
		return fmt.Errorf("simulated eject failure")
	}

	snap := c.info.Snapshot()
	if snap.Busy {
		return fmt.Errorf("cartridge is busy")
	}
	if !snap.Present {
		return fmt.Errorf("no cartridge present")
	}

	// Eject clears cartridge presence and any cartridge-specific metadata.
	c.info.SetMounted(false)
	c.info.SetRetroPie(false, nil)
	c.info.SetPresent(false)

	// Auto re-insert after 10 seconds.
	seq := atomic.AddInt64(&c.reinsertSeq, 1)
	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		select {
		case <-c.processCtx.Done():
			return
		case <-timer.C:
		}
		if atomic.LoadInt64(&c.reinsertSeq) != seq {
			return
		}
		if err := c.ApplyScenario(c.startupScenario); err != nil {
			fmt.Fprintln(os.Stderr, "scenario reinsert error:", err)
		}
	}()

	return nil
}

func (c *SimControl) Flash(ctx context.Context, reader io.Reader) error {
	if !c.info.Snapshot().Present {
		return fmt.Errorf("no cartridge present")
	}

	faults := c.Faults()
	if faults.FlashFailAfterBytes < 0 {
		return fmt.Errorf("simulated flash failure")
	}

	c.info.SetBusy(true)
	defer c.info.SetBusy(false)

	tmpPath := filepath.Join(c.root, ".sim-flash-"+fmt.Sprint(time.Now().UnixNano())+".img")
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close(); _ = os.Remove(tmpPath) }()

	buf := make([]byte, 256*1024)
	var total int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			total += int64(n)
			if faults.FlashFailAfterBytes > 0 && total >= faults.FlashFailAfterBytes {
				return fmt.Errorf("simulated flash failure after %d bytes", faults.FlashFailAfterBytes)
			}
			time.Sleep(10 * time.Millisecond)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	c.info.SetMounted(false)
	return nil
}

type SimCartridgeMounter struct {
	Control *SimControl
}

func (m SimCartridgeMounter) EnsureMounted(ctx context.Context) error {
	_ = ctx
	if m.Control == nil || m.Control.info == nil {
		return fmt.Errorf("simulator mounter not configured")
	}
	if m.Control.Faults().MountFail {
		return fmt.Errorf("simulated mount failure")
	}

	snap := m.Control.info.Snapshot()
	if !snap.Present {
		return fmt.Errorf("no cartridge present")
	}
	if snap.Mounted {
		return nil
	}
	m.Control.info.SetMounted(true)
	return nil
}

func registerSimEndpoints(handler http.Handler, control *SimControl) {
	mux, ok := handler.(*http.ServeMux)
	if !ok {
		// Only supported when the simulator uses the default mux.
		return
	}

	mux.HandleFunc("/sim/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeSimError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if err := control.Reset(); err != nil {
			writeSimError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeSimJSON(w, http.StatusOK, map[string]any{"ok": true, "scenario": control.currentScenario.Load()})
	})

	mux.HandleFunc("/sim/scenario/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeSimError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/sim/scenario/")
		name = strings.Trim(name, "/")
		if err := control.ApplyScenario(name); err != nil {
			writeSimError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeSimJSON(w, http.StatusOK, map[string]any{"ok": true, "scenario": control.currentScenario.Load()})
	})

	mux.HandleFunc("/sim/faults", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeSimJSON(w, http.StatusOK, control.Faults())
			return
		case http.MethodPost:
			var patch struct {
				MountFail           *bool  `json:"mountFail"`
				EjectFail           *bool  `json:"ejectFail"`
				FlashFailAfterBytes *int64 `json:"flashFailAfterBytes"`
			}
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				writeSimError(w, http.StatusBadRequest, "invalid json")
				return
			}
			current := control.Faults()
			if patch.MountFail != nil {
				current.MountFail = *patch.MountFail
			}
			if patch.EjectFail != nil {
				current.EjectFail = *patch.EjectFail
			}
			if patch.FlashFailAfterBytes != nil {
				current.FlashFailAfterBytes = *patch.FlashFailAfterBytes
			}
			control.SetFaults(current)
			writeSimJSON(w, http.StatusOK, current)
			return
		default:
			writeSimError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
	})
}

func writeSimJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSimError(w http.ResponseWriter, status int, message string) {
	writeSimJSON(w, status, map[string]any{"error": message})
}
