#!/usr/bin/env bash
set -euo pipefail

# NetworkManager network info helper.
#
# Usage:
#   ./netinfo.sh wifi-ssid
#   ./netinfo.sh wifi-ip
#   ./netinfo.sh ethernet-ip
#
# Output rules:
# - Print the requested value to stdout.
# - If the relevant interface is not connected/active, print nothing.

usage() {
  echo "Usage: $0 wifi-ssid|wifi-ip|ethernet-ip" >&2
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "netinfo: $1 not found in PATH" >&2
    exit 127
  }
}

wifi_iface() {
  nmcli -t -f DEVICE,TYPE device status | awk -F: '$2=="wifi"{print $1; exit 0}'
}

ethernet_iface() {
  nmcli -t -f DEVICE,TYPE device status | awk -F: '$2=="ethernet"{print $1; exit 0}'
}

device_state() {
  local dev="$1"
  nmcli -t -f DEVICE,STATE device status | awk -F: -v d="$dev" '$1==d{print $2; exit 0}'
}

active_connection_for_device() {
  local dev="$1"
  nmcli -t -f DEVICE,STATE,CONNECTION device status | awk -F: -v d="$dev" '$1==d && $2=="connected"{print $3; exit 0}'
}

ssid_for_connection() {
  local conn="$1"
  [[ -n "$conn" ]] || return 0
  nmcli -g 802-11-wireless.ssid connection show "$conn" 2>/dev/null | head -n1 | sed 's/[[:space:]]\+$//'
}

ipv4_for_device() {
  local dev="$1"
  [[ -n "$dev" ]] || return 0

  if [[ "$(device_state "$dev" || true)" != "connected" ]]; then
    return 0
  fi

  nmcli -g IP4.ADDRESS device show "$dev" 2>/dev/null | head -n1 | cut -d/ -f1 | sed 's/[[:space:]]\+$//'
}

main() {
  need_cmd nmcli
  need_cmd awk
  need_cmd sed
  need_cmd head
  need_cmd cut

  [[ ${#} -eq 1 ]] || { usage; exit 1; }

  local mode="$1"
  case "$mode" in
    wifi-ssid)
      local wdev
      wdev="$(wifi_iface || true)"
      [[ -n "$wdev" ]] || exit 0
      [[ "$(device_state "$wdev" || true)" == "connected" ]] || exit 0

      local conn
      conn="$(active_connection_for_device "$wdev" || true)"
      [[ -n "$conn" && "$conn" != "--" ]] || exit 0

      ssid_for_connection "$conn" || true
      ;;
    wifi-ip)
      ipv4_for_device "$(wifi_iface || true)" || true
      ;;
    ethernet-ip)
      ipv4_for_device "$(ethernet_iface || true)" || true
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
