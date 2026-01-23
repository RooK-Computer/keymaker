#!/usr/bin/env bash
set -euo pipefail

# Check whether the cartridge mountpoint is currently mounted.
# Silent: exit 0 if mounted, 1 if not mounted.

mountpoint="/cartridge"

[[ -d "$mountpoint" ]] || exit 1

# NOTE: findmnt --target reports the filesystem *containing* the path. If /cartridge
# is just a directory on / (unmounted), it will report the root filesystem and
# return exit 0. We need to check whether /cartridge is a mount point itself.

if command -v mountpoint >/dev/null 2>&1; then
  mountpoint -q "$mountpoint" && exit 0
  exit 1
fi

# util-linux findmnt supports -M/--mountpoint to match the mountpoint itself.
if findmnt -n -M "$mountpoint" >/dev/null 2>&1; then
  exit 0
fi

# Fallback: parse /proc/self/mounts (second field is mountpoint).
awk -v mp="$mountpoint" '$2==mp {found=1} END{exit found?0:1}' /proc/self/mounts
