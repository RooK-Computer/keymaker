#!/usr/bin/env bash
set -euo pipefail

# Mount the SD card root partition to /cartridge.
# Root partition selection: largest ext4 partition that contains /etc/fstab; if multiple, pick largest.
# Usage: sudo ./mount_sd.sh
# Silent: exit 0 on success, non-zero on failure.

mountpoint="/cartridge"

root_src=$(findmnt -n -o SOURCE / || true)
root_base="${root_src#/dev/}"
root_base="${root_base%%p*}"

target_dev="${CARTRIDGE_DEV:-}"

list_mmc() { lsblk -dn -o NAME,TYPE | awk '$2=="disk"{print $1}' | grep -E '^mmcblk[0-9]$' || true; }

if [[ -z "$target_dev" ]]; then
  for d in $(list_mmc); do
    if [[ "$d" != "$root_base" ]] && [[ -b "/dev/$d" ]]; then
      target_dev="$d"; break
    fi
  done
fi

[[ -n "$target_dev" ]] || exit 2

# Gather ext4 partitions with byte sizes
mapfile -t parts < <(lsblk -b -rno NAME,SIZE,FSTYPE "/dev/${target_dev}" | tail -n +2)
[[ ${#parts[@]} -gt 0 ]] || exit 3

# Build candidate list: NAME SIZE for ext4, then sort by SIZE desc
mapfile -t candidates < <(printf '%s\n' "${parts[@]}" | awk '$3=="ext4"{print $1" "$2}' | sort -k2,2nr)
[[ ${#candidates[@]} -gt 0 ]] || exit 4

# Iterate candidates largest to smallest; pick the first with /etc/fstab
selected_part=""
tmpmp=""
for line in "${candidates[@]}"; do
  name=$(awk '{print $1}' <<< "$line")
  devpath="/dev/${name}"
  tmpmp=$(mktemp -d)
  mount -o ro "$devpath" "$tmpmp" || { rm -rf "$tmpmp"; continue; }
  if [[ -f "$tmpmp/etc/fstab" ]]; then
    selected_part="$name"
    umount "$tmpmp"
    rm -rf "$tmpmp"
    break
  fi
  umount "$tmpmp" || true
  rm -rf "$tmpmp"
done

[[ -n "$selected_part" ]] || exit 5

devpath="/dev/${selected_part}"

# Create mountpoint if absent
mkdir -p "$mountpoint"

# Mount read-write to /cartridge
mount -o rw,relatime "$devpath" "$mountpoint"

exit 0
