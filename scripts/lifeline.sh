#!/usr/bin/env bash
set -euo pipefail

# Enable/disable the external "lifeline" circuit.
# Usage: sudo ./lifeline.sh on|off
#
# The target environment is minimal; gpioset (libgpiod) is expected to exist.
# This script must NOT block.

usage() {
  echo "Usage: $0 on|off" >&2
  echo "Environment overrides:" >&2
  echo "  LIFELINE_GPIOCHIP   (default: gpiochip0)" >&2
  echo "  LIFELINE_GPIOLINE   (default: 7)" >&2
}

[[ ${#} -eq 1 ]] || { usage; exit 1; }

cmd="$1"
value=""
case "$cmd" in
  on)  value=0 ;;  # active-low
  off) value=1 ;;
  *) usage; exit 1 ;;
esac

chip="${LIFELINE_GPIOCHIP:-gpiochip0}"
line="${LIFELINE_GPIOLINE:-7}"

command -v gpioset >/dev/null 2>&1 || {
  echo "lifeline: gpioset not found in PATH" >&2
  exit 127
}

# Prefer a non-blocking mode. On libgpiod gpioset this is typically `--mode=exit` (or `-m exit`).
# Try a few variants for compatibility; never intentionally block.
if gpioset --mode=exit "$chip" "${line}=${value}" 2>/dev/null; then
  exit 0
fi

if gpioset -m exit "$chip" "${line}=${value}" 2>/dev/null; then
  exit 0
fi

echo "lifeline: failed to set ${chip} line ${line}=${value} (gpioset)" >&2
exit 2
