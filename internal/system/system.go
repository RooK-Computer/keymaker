package system

import "context"

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
