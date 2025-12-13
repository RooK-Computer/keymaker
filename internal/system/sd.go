package system

import (
    "context"
    "fmt"
)

const (
    ejectScript          = "eject_sd.sh"
    waitForEjectScript   = "wait_for_sd_eject.sh"
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
