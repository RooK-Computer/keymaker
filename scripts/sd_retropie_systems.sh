#!/usr/bin/env bash
set -euo pipefail

# Print newline-separated list of ROM system directories under RetroPie.
# Exclude non-directories and hidden entries. Empty directories are included.
# Exit 0 on success; non-zero if root path missing.

romroot="/cartridge/home/pi/RetroPie/roms"
[[ -d "$romroot" ]] || exit 1

find "$romroot" -maxdepth 1 -mindepth 1 -type d -printf '%f\n' | grep -vE '^\.' || true

exit 0
