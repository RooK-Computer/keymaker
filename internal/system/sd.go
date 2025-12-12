package system

import (
    "context"
    "fmt"
    "path/filepath"
)

const (
    scriptsDir           = "/home/pi/Documents/rook_cartridge_writer_assistant/scripts"
    ejectScript          = "eject_sd.sh"
    waitForEjectScript   = "wait_for_sd_eject.sh"
)

// StartEject calls the eject script via sudo to initiate ejection.
func StartEject(ctx context.Context, r Runner) error {
    cmd := "sudo"
    scriptPath := filepath.Join(scriptsDir, ejectScript)
    _, stderr, err := r.Run(ctx, cmd, scriptPath)
    if err != nil {
        return fmt.Errorf("eject failed: %v: %s", err, stderr)
    }
    return nil
}

// WaitForEject calls the wait script via sudo with a timeout in seconds.
// Returns nil if ejected, or error if timeout or invocation error.
func WaitForEject(ctx context.Context, r Runner, timeoutSeconds int) error {
    cmd := "sudo"
    scriptPath := filepath.Join(scriptsDir, waitForEjectScript)
    _, stderr, err := r.Run(ctx, cmd, scriptPath, fmt.Sprintf("%d", timeoutSeconds))
    if err != nil {
        return fmt.Errorf("wait for eject failed: %v: %s", err, stderr)
    }
    return nil
}
