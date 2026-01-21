#!/usr/bin/env bash
set -euo pipefail

# Unmount the cartridge mountpoint (/cartridge) if mounted.
# Usage: sudo ./unmount_sd.sh
# Silent: exit 0 on success (also if already unmounted), non-zero on failure.

mountpoint="/cartridge"

if findmnt -n --target "$mountpoint" >/dev/null 2>&1; then
  umount "$mountpoint"
fi

exit 0
