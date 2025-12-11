#!/usr/bin/env bash
set -euo pipefail

# Wait until the internal SD mmc device disappears and the kernel logs removal.
# Usage: ./wait_for_sd_eject.sh <timeout_seconds>
# Exit 0 on kernel-confirmed removal, 2 on timeout.

timeout=${1:-60}

# Pattern that matches kernel log lines for MMC/SD card removal
pattern='mmc[^:]*:.*(card|sd).*removed'

# We'll follow only NEW kernel messages:
# - Prefer journalctl: `journalctl -k -f --since "now"`
# - Fallback to dmesg: inject a boundary marker, then `dmesg --follow` and only start matching after the marker.

tmpfifo=$(mktemp -u)
mkfifo "$tmpfifo"
trap 'rm -f "$tmpfifo"' EXIT

watch_kernel() {
  if command -v journalctl >/dev/null 2>&1; then
    # Follow new kernel logs only (since now) and stream to FIFO
    journalctl -k -f --since "now" 2>/dev/null | grep -E -m1 "$pattern" >"$tmpfifo"
  else
    # dmesg fallback: write a boundary marker to the ring buffer, then follow.
    # Note: `dmesg -n` changes console loglevel, so avoid it.
    # `dmesg --color=never` is implicit; we rely on a unique marker string.
    marker="ROOK_DMESG_BOUNDARY_$(date +%s)_$$"
    # Log the marker into the kernel ring buffer (requires CAP_SYSLOG on some systems).
    echo "$marker" | sudo -n tee /dev/kmsg >/dev/null 2>&1 || true
    # Follow kernel ring buffer; only start matching once we've seen the marker.
    dmesg --follow 2>/dev/null |
      awk -v m="$marker" -v pat="$pattern" 'BEGIN{seen=0} {
        if (!seen) {
          if (index($0, m)) { seen=1; next }
          next
        }
        if ($0 ~ pat) { print; exit }
      }' >"$tmpfifo"
  fi
}

watch_kernel &
wk_pid=$!

received=1
if command -v timeout >/dev/null 2>&1; then
  timeout "${timeout}s" head -n1 "$tmpfifo" >/dev/null 2>&1 && received=0 || received=$?
else
  end=$(( $(date +%s) + timeout ))
  while (( $(date +%s) < end )); do
    if read -r -t 1 _ < "$tmpfifo"; then received=0; break; fi
  done
fi

kill "$wk_pid" 2>/dev/null || true
rm -f "$tmpfifo"
trap - EXIT

if [[ $received -eq 0 ]]; then
  exit 0
fi

exit 2
