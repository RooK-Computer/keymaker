package system

import (
    "bytes"
    "context"
    "fmt"
    "os/exec"
)

type Runner interface {
    Run(ctx context.Context, cmd string, args ...string) (stdout, stderr string, err error)
}

type NetInfo interface {
    IP(ctx context.Context) (string, error)
}

type DeviceDetector interface {
    Detect(ctx context.Context) (path string, err error)
}

type WifiConfigurator interface {
    Configure(ctx context.Context, ssid, password string) error
}

type NoopRunner struct{}
func (NoopRunner) Run(ctx context.Context, cmd string, args ...string) (string, string, error) { return "", "", nil }

type NoopNetInfo struct{}
func (NoopNetInfo) IP(ctx context.Context) (string, error) { return "", nil }

type NoopDeviceDetector struct{}
func (NoopDeviceDetector) Detect(ctx context.Context) (string, error) { return "", nil }

type NoopWifiConfigurator struct{}
func (NoopWifiConfigurator) Configure(ctx context.Context, ssid, password string) error { return nil }

// ShellRunner executes commands via sudo and uses PATH to resolve scripts.
// It returns stdout, stderr, and an error if the command exits non-zero.
type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, cmd string, args ...string) (string, string, error) {
    // Prepend cmd to args, execute through sudo
    fullArgs := append([]string{cmd}, args...)
    c := exec.CommandContext(ctx, "sudo", fullArgs...)
    var outBuf, errBuf bytes.Buffer
    c.Stdout = &outBuf
    c.Stderr = &errBuf
    err := c.Run()
    if err != nil {
        // Include exit status if available
        if exitErr, ok := err.(*exec.ExitError); ok {
            return outBuf.String(), errBuf.String(), fmt.Errorf("exit %d: %w", exitErr.ExitCode(), err)
        }
        return outBuf.String(), errBuf.String(), err
    }
    return outBuf.String(), errBuf.String(), nil
}
