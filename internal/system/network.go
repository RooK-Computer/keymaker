package system

import (
	"context"
	"fmt"
	"strings"
)

const (
	netInfoScript = "netinfo.sh"
	wifiScript    = "wifi.sh"
)

func WiFiIPv4(ctx context.Context, r Runner) (string, error) {
	stdout, stderr, err := r.Run(ctx, netInfoScript, "wifi-ip")
	if err != nil {
		return "", fmt.Errorf("netinfo wifi-ip failed: %v: %s", err, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func WiFiSSID(ctx context.Context, r Runner) (string, error) {
	stdout, stderr, err := r.Run(ctx, netInfoScript, "wifi-ssid")
	if err != nil {
		return "", fmt.Errorf("netinfo wifi-ssid failed: %v: %s", err, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func EthernetIPv4(ctx context.Context, r Runner) (string, error) {
	stdout, stderr, err := r.Run(ctx, netInfoScript, "ethernet-ip")
	if err != nil {
		return "", fmt.Errorf("netinfo ethernet-ip failed: %v: %s", err, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func EnsureHotspot(ctx context.Context, r Runner) error {
	_, stderr, err := r.Run(ctx, wifiScript, "hotspot")
	if err != nil {
		return fmt.Errorf("wifi hotspot failed: %v: %s", err, stderr)
	}
	return nil
}

func JoinWiFi(ctx context.Context, r Runner, ssid, password string) error {
	ssid = strings.TrimSpace(ssid)
	password = strings.TrimSpace(password)
	if ssid == "" {
		return fmt.Errorf("wifi join failed: empty ssid")
	}
	_, stderr, err := r.Run(ctx, wifiScript, "join", ssid, password)
	if err != nil {
		return fmt.Errorf("wifi join failed: %v: %s", err, stderr)
	}
	return nil
}
