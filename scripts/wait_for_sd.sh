#!/usr/bin/env bash
set -euo pipefail

# Wait until the internal SD mmc device appears.
# Usage: ./wait_for_sd.sh <timeout_seconds>
# Exit 0 on appearance, 2 on timeout, 1 on other errors.

timeout=${1:-60}

root_src=$(findmnt -n -o SOURCE / || true)
root_base="${root_src#/dev/}"
root_base="${root_base%%p*}"

list_mmc() { lsblk -dn -o NAME,TYPE | awk '$2=="disk"{print $1}' | grep -E '^mmcblk[0-9]$' || true; }

start=$(date +%s)
while true; do
  now=$(date +%s)
  if (( now - start >= timeout )); then exit 2; fi
  for d in $(list_mmc); do
    [[ "$d" == "$root_base" ]] && continue
    # appearance detected
    exit 0
  done
  sleep 0.5
done
