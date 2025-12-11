#!/usr/bin/env bash
set -euo pipefail

# Unmount and unbind the internal SD card mmc device so it disappears.
# Usage: sudo ./eject_sd.sh [-f]
# -f : force unmount (lazy) if busy
# Silent: exit 0 success, non-zero failure.

force_unmount=0
while getopts ":f" opt; do
  case "$opt" in
    f) force_unmount=1 ;;
    *) ;;
  esac
done

# Determine target mmc device (not the root one)
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
[[ -b "/dev/${target_dev}" ]] || exit 3

# Unmount all partitions for target_dev
parts=$(lsblk -rno NAME "/dev/${target_dev}" | tail -n +2 || true)
if [[ -n "$parts" ]]; then
  while read -r p; do
    mp=$(findmnt -n -o TARGET "/dev/${p}" || true)
    if [[ -n "$mp" ]]; then
      if [[ $force_unmount -eq 1 ]]; then
        umount -l "$mp" || true
      else
        umount "$mp"
      fi
    fi
  done <<< "$parts"
fi

#!/usr/bin/env bash
set -euo pipefail

# After unmounting, explicitly unbind via /sys/bus/mmc/drivers/mmcblk/unbind using the device identifier.

# Identify mmc sys device: /sys/block/mmcblkX/device
sysdev="/sys/block/${target_dev}/device"
[[ -d "$sysdev" ]] || exit 4
sysdev=$(readlink "$sysdev")

# Determine the device identifier expected by mmcblk driver unbind.
# For mmcblk, the device identifier corresponds to the mmc card device under mmc host, e.g., mmcX:0001
card_node=$(basename "$sysdev")
# card_node is typically mmcX:0001; validate format
[[ -n "$card_node" ]] || exit 4

unbind_path="/sys/bus/mmc/drivers/mmcblk/unbind"
[[ -w "$unbind_path" ]] || exit 4

echo "$card_node" >> "$unbind_path"

# Verify disappearance
sleep 0.5
[[ -b "/dev/${target_dev}" ]] && exit 5

exit 0
