#!/usr/bin/env bash
set -euo pipefail

# Check whether the cartridge mountpoint is currently mounted.
# Silent: exit 0 if mounted, 1 if not mounted.

mountpoint="/cartridge"

if findmnt -n --target "$mountpoint" >/dev/null 2>&1; then
  exit 0
fi

exit 1
