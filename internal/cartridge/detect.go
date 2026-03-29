package cartridge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rook-computer/keymaker/internal/state"
	"github.com/rook-computer/keymaker/internal/system"
)

const retroPieRomsRoot = "/cartridge/home/pi/RetroPie/roms"

type Logger interface {
	Infof(component string, format string, args ...interface{})
	Errorf(component string, format string, args ...interface{})
}

type DetectOptions struct {
	// ManageBusy toggles CartridgeInfo.Busy while detecting.
	ManageBusy bool

	// Retries controls how often detection is retried when mounting fails.
	Retries int

	// RetryDelay is used between retries.
	RetryDelay time.Duration
}

func DetectAndUpdate(ctx context.Context, runner system.Runner, logger Logger, opts DetectOptions) error {
	if runner == nil {
		return errors.New("no system runner configured")
	}
	if opts.Retries <= 0 {
		opts.Retries = 1
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = 750 * time.Millisecond
	}

	cartridgeInfo := state.GetCartridgeInfo()
	if opts.ManageBusy {
		cartridgeInfo.SetBusy(true)
		defer cartridgeInfo.SetBusy(false)
	}

	present, err := system.IsCartridgePresent(ctx, runner)
	if err != nil {
		if logger != nil {
			logger.Errorf("system", "present detection failed: %v", err)
		}
	}
	if !present {
		cartridgeInfo.Reset()
		return nil
	}

	cartridgeInfo.SetPresent(true)

	mountedBefore, err := system.IsCartridgeMounted(ctx, runner)
	if err != nil {
		if logger != nil {
			logger.Errorf("system", "mount detection failed: %v", err)
		}
	}

	mountedNow := mountedBefore
	if !mountedBefore {
		for attempt := 0; attempt < opts.Retries; attempt++ {
			if err := system.MountCartridge(ctx, runner); err != nil {
				if logger != nil {
					logger.Errorf("system", "mount failed (attempt %d/%d): %v", attempt+1, opts.Retries, err)
				}
				select {
				case <-ctx.Done():
					cartridgeInfo.SetMounted(false)
					return ctx.Err()
				case <-time.After(opts.RetryDelay):
					continue
				}
			}
			mountedNow = true
			break
		}
	}
	cartridgeInfo.SetMounted(mountedNow)

	isRetroPie := false
	if mountedNow {
		isRetroPie, err = system.IsRetroPieCartridge(ctx, runner)
		if err != nil {
			if logger != nil {
				logger.Errorf("system", "retropie check failed: %v", err)
			}
			isRetroPie = false
		}
	}

	var systemsWithFiles []state.CartridgeSystemInfo
	var emptySystems []string
	if isRetroPie {
		var detectedSystems []string
		detectedSystems, err = system.RetroPieSystems(ctx, runner)
		if err != nil {
			// Per implementation plan: if systems fail, overrule and treat as not RetroPie.
			if logger != nil {
				logger.Errorf("system", "retropie systems failed, treating as non-retropie: %v", err)
			}
			isRetroPie = false
			systemsWithFiles = nil
			emptySystems = nil
		} else {
			systemsWithFiles, emptySystems, err = collectRetroPieSystemInfo(retroPieRomsRoot, detectedSystems)
			if err != nil {
				return err
			}
		}
	}
	cartridgeInfo.SetRetroPie(isRetroPie, systemsWithFiles, emptySystems)

	// If the cartridge wasn't mounted before, ensure it isn't left mounted.
	if !mountedBefore {
		mountedAfter, err := system.IsCartridgeMounted(ctx, runner)
		if err != nil {
			if logger != nil {
				logger.Errorf("system", "mount detection failed (post-analyze): %v", err)
			}
		} else if mountedAfter {
			if err := system.UnmountCartridge(ctx, runner); err != nil {
				if logger != nil {
					logger.Errorf("system", "unmount failed: %v", err)
				}
			} else {
				cartridgeInfo.SetMounted(false)
			}
		}
	}

	return nil
}

func collectRetroPieSystemInfo(romsRoot string, detectedSystems []string) ([]state.CartridgeSystemInfo, []string, error) {
	systemsWithFiles := make([]state.CartridgeSystemInfo, 0, len(detectedSystems))
	emptySystems := make([]string, 0, len(detectedSystems))

	for _, systemName := range detectedSystems {
		visibleEntryCount, err := countVisibleEntries(filepath.Join(romsRoot, systemName))
		if err != nil {
			return nil, nil, err
		}
		if visibleEntryCount == 0 {
			emptySystems = append(emptySystems, systemName)
			continue
		}
		systemsWithFiles = append(systemsWithFiles, state.CartridgeSystemInfo{
			System:    systemName,
			FileCount: visibleEntryCount,
		})
	}

	sort.Slice(systemsWithFiles, func(leftIndex, rightIndex int) bool {
		return systemsWithFiles[leftIndex].System < systemsWithFiles[rightIndex].System
	})
	sort.Strings(emptySystems)

	return systemsWithFiles, emptySystems, nil
}

func countVisibleEntries(directoryPath string) (int, error) {
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return 0, err
	}

	visibleEntryCount := 0
	for _, entry := range entries {
		entryName := strings.TrimSpace(entry.Name())
		if entryName == "" {
			continue
		}
		if strings.HasPrefix(entryName, ".") {
			continue
		}
		visibleEntryCount++
	}

	return visibleEntryCount, nil
}
