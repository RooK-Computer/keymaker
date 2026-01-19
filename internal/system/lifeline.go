package system

import (
	"context"
	"fmt"
)

const lifelineScript = "lifeline.sh"

func LifelineOn(ctx context.Context, r Runner) error {
	return lifeline(ctx, r, "on")
}

func LifelineOff(ctx context.Context, r Runner) error {
	return lifeline(ctx, r, "off")
}

func lifeline(ctx context.Context, r Runner, mode string) error {
	_, stderr, err := r.Run(ctx, lifelineScript, mode)
	if err != nil {
		return fmt.Errorf("lifeline %s failed: %v: %s", mode, err, stderr)
	}
	return nil
}
