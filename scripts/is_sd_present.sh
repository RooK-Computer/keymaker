#!/usr/bin/env bash
set -euo pipefail

# Detect if the removable SD card (internal mmc slot) is present as a block device.
# Silent: exit 0 if present, 1 if absent.

# Optionally allow override via CARTRIDGE_DEV env, e.g., CARTRIDGE_DEV=mmcblk0
target_dev="${CARTRIDGE_DEV:-}"

root_dev=$(findmnt -n -o SOURCE / || true)

list_mmc() {
  lsblk -dn -o NAME,TYPE | awk '$2=="disk"{print $1}' | grep -E '^mmcblk[0-9]$' || true
}

if [[ -n "$target_dev" ]]; then
  [[ -b "/dev/${target_dev}" ]] && exit 0 || exit 1
fi

# Prefer any mmc block device that is not the current root device (to avoid internal/boot confusion)
for dev in $(list_mmc); do
  if [[ -b "/dev/${dev}" ]]; then
    # If root is mmcblkXpY, extract base
    root_base="${root_dev#/dev/}"
    root_base="${root_base%%p*}"
    if [[ "${dev}" != "${root_base}" ]]; then
      exit 0
    fi
  fi
done

exit 1
