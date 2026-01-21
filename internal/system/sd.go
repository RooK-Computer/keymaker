package system

import (
	"context"
	"fmt"
	"strings"
)

const (
	ejectScript            = "eject_sd.sh"
	waitForEjectScript     = "wait_for_sd_eject.sh"
	waitForInsertScript    = "wait_for_sd.sh"
	mountCartridgeScript   = "mount_sd.sh"
	unmountCartridgeScript = "unmount_sd.sh"
	isMountedScript        = "is_sd_mounted.sh"
	isRetroPieScript       = "is_sd_retropie.sh"
	retroPieSystemsScript  = "sd_retropie_systems.sh"
)

// StartEject calls the eject script via sudo to initiate ejection.
func StartEject(ctx context.Context, r Runner) error {
	cmd := ejectScript
	_, stderr, err := r.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("eject failed: %v: %s", err, stderr)
	}
	return nil
}

// WaitForEject calls the wait script via sudo with a timeout in seconds.
// Returns nil if ejected, or error if timeout or invocation error.
func WaitForEject(ctx context.Context, r Runner, timeoutSeconds int) error {
	cmd := waitForEjectScript
	_, stderr, err := r.Run(ctx, cmd, fmt.Sprintf("%d", timeoutSeconds))
	if err != nil {
		return fmt.Errorf("wait for eject failed: %v: %s", err, stderr)
	}
	return nil
}

// WaitForInsert waits until a cartridge SD appears.
func WaitForInsert(ctx context.Context, r Runner, timeoutSeconds int) error {
	cmd := waitForInsertScript
	_, stderr, err := r.Run(ctx, cmd, fmt.Sprintf("%d", timeoutSeconds))
	if err != nil {
		return fmt.Errorf("wait for insert failed: %v: %s", err, stderr)
	}
	return nil
}

// MountCartridge mounts the cartridge to /cartridge.
func MountCartridge(ctx context.Context, r Runner) error {
	cmd := mountCartridgeScript
	_, stderr, err := r.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("mount cartridge failed: %v: %s", err, stderr)
	}
	return nil
}

// UnmountCartridge unmounts /cartridge if mounted.
func UnmountCartridge(ctx context.Context, r Runner) error {
	cmd := unmountCartridgeScript
	_, stderr, err := r.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("unmount cartridge failed: %v: %s", err, stderr)
	}
	return nil
}

func IsCartridgeMounted(ctx context.Context, r Runner) (bool, error) {
	// Non-zero exit means "not mounted".
	_, _, err := r.Run(ctx, isMountedScript)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// IsRetroPieCartridge checks whether the mounted cartridge looks like a RetroPie install.
// Any non-zero exit code is treated as "not RetroPie".
func IsRetroPieCartridge(ctx context.Context, r Runner) (bool, error) {
	_, _, err := r.Run(ctx, isRetroPieScript)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// RetroPieSystems returns the newline-separated list of systems reported by the script.
func RetroPieSystems(ctx context.Context, r Runner) ([]string, error) {
	stdout, stderr, err := r.Run(ctx, retroPieSystemsScript)
	if err != nil {
		return nil, fmt.Errorf("retropie systems failed: %v: %s", err, stderr)
	}
	stdout = strings.ReplaceAll(stdout, "\r\n", "\n")
	lines := strings.Split(stdout, "\n")
	var systems []string
	for _, line := range lines {
		systemName := strings.TrimSpace(line)
		if systemName == "" {
			continue
		}
		systems = append(systems, systemName)
	}
	return systems, nil
}
