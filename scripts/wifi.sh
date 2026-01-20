#!/usr/bin/env bash
set -euo pipefail

# NetworkManager WiFi helper.
#
# Modes:
#   hotspot
#     Ensure an open hotspot exists and is active.
#     SSID format: RooK-<4 digits>
#     IPv4 subnet: 192.168.0.0/24 (gateway: 192.168.0.1)
#
#   surveillance
#     Block and stream visible WiFi SSIDs line-by-line.
#     Intended to be killed; must not disrupt the current WiFi state.
#
#   join <ssid> <password>
#     Join an existing WiFi network; updates a known connection when possible.
#
# Usage:
#   sudo ./wifi.sh hotspot
#   sudo ./wifi.sh surveillance
#   sudo ./wifi.sh join "MyWifi" "secretpassword"
#
# Environment overrides:
#   WIFI_IFACE            (default: auto-detect first WiFi device)
#   HOTSPOT_CONN_NAME     (default: rook-hotspot)
#   HOTSPOT_ADDR          (default: 192.168.0.1/24)
#   HOTSPOT_SSID_PREFIX   (default: RooK-)
#   SURVEILLANCE_INTERVAL (default: 2)

usage() {
  echo "Usage:" >&2
  echo "  $0 hotspot" >&2
  echo "  $0 surveillance" >&2
  echo "  $0 join <ssid> <password>" >&2
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "wifi: $1 not found in PATH" >&2
    exit 127
  }
}

wifi_iface() {
  if [[ -n "${WIFI_IFACE:-}" ]]; then
    echo "$WIFI_IFACE"
    return 0
  fi

  # Prefer an actual WiFi device.
  nmcli -t -f DEVICE,TYPE device status | awk -F: '$2=="wifi"{print $1; exit 0}'
}

rand4() {
  # 0000-9999
  local n
  n=$(od -An -N2 -tu2 /dev/urandom | tr -d ' ')
  printf "%04d" "$((n % 10000))"
}

conn_exists() {
  local name="$1"
  nmcli -t -f NAME connection show | grep -Fxq "$name"
}

conn_type() {
  local name="$1"
  nmcli -t -f connection.type connection show "$name" 2>/dev/null || true
}

conn_ssid() {
  local name="$1"
  nmcli -g 802-11-wireless.ssid connection show "$name" 2>/dev/null || true
}

ensure_hotspot_conn() {
  local iface="$1"
  local conn_name="$2"
  local hotspot_addr="$3"
  local ssid_prefix="$4"

  if conn_exists "$conn_name"; then
    return 0
  fi

  local ssid
  ssid="${ssid_prefix}$(rand4)"

  # Create an open AP with shared IPv4 and a fixed private subnet.
  nmcli connection add \
    type wifi \
    ifname "$iface" \
    con-name "$conn_name" \
    ssid "$ssid" \
    802-11-wireless.mode ap \
    802-11-wireless.band bg \
    ipv4.method shared \
    ipv4.addresses "$hotspot_addr" \
    ipv6.method ignore \
    >/dev/null

  # Ensure it is unencrypted.
  nmcli connection modify "$conn_name" 802-11-wireless-security.key-mgmt none >/dev/null

  # Make it resilient across boots.
  nmcli connection modify "$conn_name" connection.autoconnect yes >/dev/null
}

hotspot_up() {
  local iface="$1"
  local conn_name="$2"

  nmcli radio wifi on >/dev/null || true

  # Disconnect to reduce flakiness when switching from another active connection.
  nmcli device disconnect "$iface" >/dev/null 2>&1 || true

  nmcli connection up "$conn_name" >/dev/null
}

hotspot_down_if_active() {
  local conn_name="$1"
  if nmcli -t -f NAME connection show --active | grep -Fxq "$conn_name"; then
    nmcli connection down "$conn_name" >/dev/null || true
  fi
}

surveillance() {
  local interval="$1"

  # Dedupe across scans so consumers get a stable stream.
  declare -A seen

  trap 'exit 0' INT TERM

  while true; do
    # NOTE: --rescan yes can block briefly; keep it simple and robust.
    while IFS= read -r ssid; do
      [[ -n "$ssid" ]] || continue
      if [[ -z "${seen[$ssid]+x}" ]]; then
        seen["$ssid"]=1
        printf '%s\n' "$ssid"
      fi
    done < <(nmcli -t -f SSID device wifi list --rescan yes | sed 's/[[:space:]]\+$//' | grep -v '^--$' || true)

    sleep "$interval"
  done
}

join_network() {
  local iface="$1"
  local hotspot_conn_name="$2"
  local ssid="$3"
  local password="$4"

  nmcli radio wifi on >/dev/null || true

  # Stop hotspot if it is currently active.
  hotspot_down_if_active "$hotspot_conn_name"

  # If there's a connection whose name equals the SSID, update it.
  if conn_exists "$ssid" && [[ "$(conn_type "$ssid")" == "802-11-wireless" ]]; then
    if [[ -n "$password" ]]; then
      nmcli connection modify "$ssid" 802-11-wireless-security.psk "$password" >/dev/null
      nmcli connection modify "$ssid" 802-11-wireless-security.key-mgmt wpa-psk >/dev/null || true
    fi
    nmcli connection up "$ssid" >/dev/null
    return 0
  fi

  # Otherwise, connect via scan results (will create/choose a suitable connection).
  # nmcli will name the connection after the SSID by default.
  if [[ -n "$password" ]]; then
    nmcli device wifi connect "$ssid" password "$password" ifname "$iface" >/dev/null
  else
    nmcli device wifi connect "$ssid" ifname "$iface" >/dev/null
  fi
}

main() {
  need_cmd nmcli
  need_cmd awk
  need_cmd od
  need_cmd sed
  need_cmd grep

  [[ ${#} -ge 1 ]] || { usage; exit 1; }

  local mode="$1"
  local iface
  iface="$(wifi_iface || true)"
  [[ -n "$iface" ]] || { echo "wifi: no WiFi device found" >&2; exit 2; }

  local hotspot_conn_name="${HOTSPOT_CONN_NAME:-rook-hotspot}"
  local hotspot_addr="${HOTSPOT_ADDR:-192.168.0.1/24}"
  local ssid_prefix="${HOTSPOT_SSID_PREFIX:-RooK-}"
  local surveillance_interval="${SURVEILLANCE_INTERVAL:-2}"

  case "$mode" in
    hotspot)
      ensure_hotspot_conn "$iface" "$hotspot_conn_name" "$hotspot_addr" "$ssid_prefix"
      hotspot_up "$iface" "$hotspot_conn_name"
      ;;
    surveillance)
      surveillance "$surveillance_interval"
      ;;
    join)
      [[ ${#} -eq 3 ]] || { usage; exit 1; }
      join_network "$iface" "$hotspot_conn_name" "$2" "$3"
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
