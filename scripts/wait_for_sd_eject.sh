#!/usr/bin/env bash
set -euo pipefail

# Wait until the internal SD mmc device disappears and the kernel logs removal.
# Usage: ./wait_for_sd_eject.sh <timeout_seconds>
# Exit 0 on kernel-confirmed removal, 2 on timeout, 1 on other errors.

timeout=${1:-60}
remaining=$timeout

# Wait for kernel to log removal (monitor dmesg)
pattern='mmc[^:]*:.*(card|sd).*removed'

tmpfifo=$(mktemp -u)
mkfifo "$tmpfifo"
trap 'rm -f "$tmpfifo"' EXIT

(
  dmesg -w 2>/dev/null | grep -E -m1 "$pattern" > "$tmpfifo"
) &
dm_pid=$!

received=1
if command -v timeout >/dev/null 2>&1; then
  timeout "${remaining}s" head -n1 "$tmpfifo" >/dev/null 2>&1 && received=0 || received=$?
else
  end=$(( $(date +%s) + remaining ))
  while (( $(date +%s) < end )); do
    if read -r -t 1 _ < "$tmpfifo"; then received=0; break; fi
  done
fi

kill "$dm_pid" 2>/dev/null || true
rm -f "$tmpfifo"
trap - EXIT

if [[ $received -eq 0 ]]; then
  exit 0
fi

exit 2
