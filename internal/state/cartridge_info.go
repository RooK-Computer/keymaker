package state

import "sync"

type CartridgeInfoSnapshot struct {
	Present    bool
	Mounted    bool
	IsRetroPie bool
	Systems    []string
	Busy       bool
}

type CartridgeInfo struct {
	mu sync.RWMutex

	present    bool
	mounted    bool
	isRetroPie bool
	systems    []string
	busy       bool
}

var (
	cartridgeInfoOnce sync.Once
	cartridgeInfo     *CartridgeInfo
)

func GetCartridgeInfo() *CartridgeInfo {
	cartridgeInfoOnce.Do(func() {
		cartridgeInfo = &CartridgeInfo{}
	})
	return cartridgeInfo
}

func (info *CartridgeInfo) Snapshot() CartridgeInfoSnapshot {
	info.mu.RLock()
	defer info.mu.RUnlock()

	return CartridgeInfoSnapshot{
		Present:    info.present,
		Mounted:    info.mounted,
		IsRetroPie: info.isRetroPie,
		Systems:    cloneStrings(info.systems),
		Busy:       info.busy,
	}
}

func (info *CartridgeInfo) Reset() {
	info.mu.Lock()
	info.present = false
	info.mounted = false
	info.isRetroPie = false
	info.systems = nil
	info.busy = false
	info.mu.Unlock()
}

func (info *CartridgeInfo) SetPresent(present bool) {
	info.mu.Lock()
	info.present = present
	info.mu.Unlock()
}

func (info *CartridgeInfo) SetMounted(mounted bool) {
	info.mu.Lock()
	info.mounted = mounted
	info.mu.Unlock()
}

func (info *CartridgeInfo) SetBusy(busy bool) {
	info.mu.Lock()
	info.busy = busy
	info.mu.Unlock()
}

func (info *CartridgeInfo) SetRetroPie(isRetroPie bool, systems []string) {
	info.mu.Lock()
	info.isRetroPie = isRetroPie
	if isRetroPie {
		info.systems = cloneStrings(systems)
	} else {
		info.systems = nil
	}
	info.mu.Unlock()
}

func cloneStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}
