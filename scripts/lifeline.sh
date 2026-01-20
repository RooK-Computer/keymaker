#!/usr/bin/env bash
set -euo pipefail

# Enable/disable the external "lifeline" circuit.
# Usage: sudo ./lifeline.sh on|off
#
# The target environment is minimal; gpioset (libgpiod) is expected to exist.
#
# NOTE: Newer libgpiod releases removed the old "exit" mode semantics for gpioset.
# gpioset keeps the line requested until the gpioset process exits. To both (a)
# keep the pin driven and (b) have THIS script exit immediately, we run gpioset
# in daemon/background mode.

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
consumer="${LIFELINE_CONSUMER:-keymaker-lifeline}"

command -v gpioset >/dev/null 2>&1 || {
  echo "lifeline: gpioset not found in PATH" >&2
  exit 127
}

kill_existing_gpioset() {
  # Free the GPIO line in case a prior daemonized gpioset instance is holding it.
  # We identify our own gpioset process(es) via the consumer string and ignore
  # other arguments (chip/line/value). This keeps the match simple and robust.
  local pids=""

  # Prefer a basic ps|grep|grep pipeline for portability.
  # Note: use a [g]pioset regex to avoid matching the grep process itself.
  pids="$(ps ax -o pid= -o command= 2>/dev/null | \
    grep -E "[g]pioset" | \
    grep -F "${consumer}" | \
    awk '{print $1}' || true)"

  # Fallback for very minimal ps implementations.
  if [[ -z "$pids" ]]; then
    pids="$(ps ax 2>/dev/null | \
      grep -E "[g]pioset" | \
      grep -F "${consumer}" | \
      awk '{print $1}' || true)"
  fi

  [[ -n "$pids" ]] || return 0

  # Best-effort graceful shutdown, then force.
  kill $pids 2>/dev/null || true
  for _ in 1 2 3 4 5; do
    sleep 0.1
    if kill -0 $pids 2>/dev/null; then
      continue
    fi
    return 0
  done
  kill -9 $pids 2>/dev/null || true
}

kill_existing_gpioset

# libgpiod v2+ syntax: gpioset [OPTIONS] <line=value>… (with optional -c/--chip).
# Use --daemonize so the line stays requested while this script exits.
if gpioset -c "$chip" -C "$consumer" --daemonize "${line}=${value}" 2>/dev/null; then
  exit 0
fi

# libgpiod v1 fallback syntax: gpioset [OPTIONS] <chip> <offset=value>…
# Use mode=signal with backgrounding to keep the line driven after this script returns.
if gpioset -m signal -b "$chip" "${line}=${value}" 2>/dev/null; then
  exit 0
fi

echo "lifeline: failed to set ${chip} line ${line}=${value} (gpioset)" >&2
exit 2
