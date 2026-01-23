package state

import "sync"

type WiFiMode int

const (
	WiFiModeUnknown WiFiMode = iota
	WiFiModeHotspot
	WiFiModeJoin
)

type WiFiConfigSnapshot struct {
	Initialized bool
	NeedsApply  bool
	Mode        WiFiMode
	SSID        string
	Password    string
}

type WiFiConfig struct {
	mu sync.RWMutex

	initialized bool
	needsApply  bool
	mode        WiFiMode
	ssid        string
	password    string
}

var (
	wifiConfigOnce sync.Once
	wifiConfig     *WiFiConfig
)

func GetWiFiConfig() *WiFiConfig {
	wifiConfigOnce.Do(func() {
		wifiConfig = &WiFiConfig{mode: WiFiModeUnknown}
	})
	return wifiConfig
}

func (config *WiFiConfig) Snapshot() WiFiConfigSnapshot {
	config.mu.RLock()
	defer config.mu.RUnlock()

	return WiFiConfigSnapshot{
		Initialized: config.initialized,
		NeedsApply:  config.needsApply,
		Mode:        config.mode,
		SSID:        config.ssid,
		Password:    config.password,
	}
}

func (config *WiFiConfig) Reset() {
	config.mu.Lock()
	config.initialized = false
	config.needsApply = false
	config.mode = WiFiModeUnknown
	config.ssid = ""
	config.password = ""
	config.mu.Unlock()
}

func (config *WiFiConfig) SetMode(mode WiFiMode) {
	config.mu.Lock()
	if config.mode != mode {
		config.needsApply = true
	}
	config.mode = mode
	config.initialized = true
	config.mu.Unlock()
}

func (config *WiFiConfig) SetSSID(ssid string) {
	config.mu.Lock()
	if config.ssid != ssid {
		config.needsApply = true
	}
	config.ssid = ssid
	config.initialized = true
	config.mu.Unlock()
}

func (config *WiFiConfig) SetPassword(password string) {
	config.mu.Lock()
	if config.password != password {
		config.needsApply = true
	}
	config.password = password
	config.initialized = true
	config.mu.Unlock()
}

func (config *WiFiConfig) SetHotspot() {
	config.mu.Lock()
	if !config.initialized || config.mode != WiFiModeHotspot {
		config.needsApply = true
	}
	config.mode = WiFiModeHotspot
	config.initialized = true
	config.mu.Unlock()
}

func (config *WiFiConfig) SetJoin(ssid, password string) {
	config.mu.Lock()
	if !config.initialized || config.mode != WiFiModeJoin || config.ssid != ssid || config.password != password {
		config.needsApply = true
	}
	config.mode = WiFiModeJoin
	config.ssid = ssid
	config.password = password
	config.initialized = true
	config.mu.Unlock()
}

// MarkApplied marks the current WiFiConfig as already applied to the system.
// Call this after successfully bringing up hotspot/join on the device.
func (config *WiFiConfig) MarkApplied() {
	config.mu.Lock()
	config.needsApply = false
	config.mu.Unlock()
}
