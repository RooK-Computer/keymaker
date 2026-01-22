#!/usr/bin/env bash
set -euo pipefail

# Flash a gzipped disk image to the cartridge block device.
#
# Input:  gzipped image on stdin
# Output: none (silent)
#
# Usage:
#   sudo ./flash.sh < image.img.gz
#
# Environment overrides:
#   CARTRIDGE_DEV (e.g. mmcblk0)

mountpoint="/cartridge"

# Determine root base device (to avoid self-destruction)
root_src=$(findmnt -n -o SOURCE / || true)
root_base="${root_src#/dev/}"
root_base="${root_base%%p*}"

# Pick target mmc block device
list_mmc() { lsblk -dn -o NAME,TYPE | awk '$2=="disk"{print $1}' | grep -E '^mmcblk[0-9]$' || true; }

target_dev="${CARTRIDGE_DEV:-}"
if [[ -z "$target_dev" ]]; then
  for d in $(list_mmc); do
    if [[ "$d" != "$root_base" ]] && [[ -b "/dev/$d" ]]; then
      target_dev="$d"; break
    fi
  done
fi

[[ -n "$target_dev" ]] || exit 2
[[ -b "/dev/${target_dev}" ]] || exit 3
[[ "$target_dev" != "$root_base" ]] || exit 4

# Unmount any mounted partitions for the target device (including /cartridge)
parts=$(lsblk -rno NAME "/dev/${target_dev}" | tail -n +2 || true)
if [[ -n "$parts" ]]; then
  while read -r p; do
    mp=$(findmnt -n -o TARGET "/dev/${p}" || true)
    if [[ -n "$mp" ]]; then
      umount "$mp" || true
    fi
  done <<< "$parts"
fi

# Also ensure /cartridge is unmounted, in case it is mounted by label/path
if findmnt -n --target "$mountpoint" >/dev/null 2>&1; then
  umount "$mountpoint" || true
fi

# Stream stdin -> gunzip -> dd to whole device
# conv=fsync ensures data is flushed before dd exits.
# status=none keeps the script silent.
gunzip -c | dd of="/dev/${target_dev}" bs=4M conv=fsync status=none

sync

exit 0
