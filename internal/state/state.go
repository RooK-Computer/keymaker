package state

import "sync"

type Phase int

const (
    BOOTING Phase = iota
    READY
    FLASHING
    DONE
    ERROR
    CANCELLED
)

type WiFiInfo struct {
    SSID     string
    Password string
    QRPayload string
}

type NetworkInfo struct {
    IP      string
    URL     string
    URLQR   string
}

type FlashInfo struct {
    Device       string
    BytesWritten int64
    WriteRate    int64 // bytes/sec, optional
    Status       string
    Err          string
}

type State struct {
    Phase   Phase
    WiFi    WiFiInfo
    Network NetworkInfo
    Flash   FlashInfo
}

type Store struct {
    mu    sync.RWMutex
    state State
}

func NewStore() *Store {
    return &Store{state: State{Phase: BOOTING}}
}

func (s *Store) Snapshot() State {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.state
}

func (s *Store) SetPhase(p Phase) {
    s.mu.Lock()
    s.state.Phase = p
    s.mu.Unlock()
}

func (s *Store) UpdateWiFi(w WiFiInfo) { s.mu.Lock(); s.state.WiFi = w; s.mu.Unlock() }
func (s *Store) UpdateNetwork(n NetworkInfo) { s.mu.Lock(); s.state.Network = n; s.mu.Unlock() }
func (s *Store) UpdateFlash(f FlashInfo) { s.mu.Lock(); s.state.Flash = f; s.mu.Unlock() }
