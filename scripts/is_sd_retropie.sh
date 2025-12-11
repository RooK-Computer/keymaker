#!/usr/bin/env bash
set -euo pipefail

# Check whether /cartridge contains a RetroPie install in expected paths.
# Silent: exit 0 if paths exist, non-zero otherwise.

[[ -d "/cartridge/opt/retropie" ]] && [[ -d "/cartridge/home/pi/RetroPie/roms" ]] && exit 0
exit 1
