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
	SSID      string
	Password  string
	QRPayload string
}

type NetworkInfo struct {
	IP    string
	URL   string
	URLQR string
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

func (store *Store) Snapshot() State {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.state
}

func (store *Store) SetPhase(phase Phase) {
	store.mu.Lock()
	store.state.Phase = phase
	store.mu.Unlock()
}

func (store *Store) UpdateWiFi(wifi WiFiInfo) {
	store.mu.Lock()
	store.state.WiFi = wifi
	store.mu.Unlock()
}

func (store *Store) UpdateNetwork(network NetworkInfo) {
	store.mu.Lock()
	store.state.Network = network
	store.mu.Unlock()
}

func (store *Store) UpdateFlash(flash FlashInfo) {
	store.mu.Lock()
	store.state.Flash = flash
	store.mu.Unlock()
}
