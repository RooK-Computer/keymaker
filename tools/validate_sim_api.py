#!/usr/bin/env python3

import http.client
import json
import os
import socket
import time
from urllib.parse import urlparse


def _fail(message: str) -> None:
    raise SystemExit(f"validate_sim_api: {message}")


def _request(method: str, base: str, path: str, body: bytes | None = None, headers: dict[str, str] | None = None):
    parsed = urlparse(base)
    if parsed.scheme not in ("http", "https"):
        _fail(f"unsupported base URL scheme: {base}")

    conn_cls = http.client.HTTPSConnection if parsed.scheme == "https" else http.client.HTTPConnection
    conn = conn_cls(parsed.hostname, parsed.port, timeout=5)
    try:
        conn.request(method, path, body=body, headers=headers or {})
        resp = conn.getresponse()
        data = resp.read()
        return resp.status, dict(resp.getheaders()), data
    finally:
        conn.close()


def _wait_up(base: str, timeout_s: float = 5.0) -> None:
    deadline = time.time() + timeout_s
    last_error: str | None = None
    while time.time() < deadline:
        try:
            status, _, body = _request("GET", base, "/api/v1/cartridgeinfo")
            if status == 200:
                json.loads(body.decode("utf-8"))
                return
            last_error = f"unexpected status {status}"
        except Exception as exc:  # noqa: BLE001
            last_error = str(exc)
        time.sleep(0.1)
    _fail(f"simulator did not become ready: {last_error}")


def _request_raw_chunked_status(base: str, path: str, payload: bytes) -> int:
    parsed = urlparse(base)
    if parsed.scheme != "http":
        _fail("raw chunked request only implemented for http")

    host = parsed.hostname
    port = parsed.port
    if host is None or port is None:
        _fail(f"invalid base URL: {base}")

    chunk_size = format(len(payload), "x").encode("ascii")
    req = b"".join(
        [
            f"POST {path} HTTP/1.1\r\n".encode("ascii"),
            f"Host: {host}:{port}\r\n".encode("ascii"),
            b"Content-Type: application/octet-stream\r\n",
            b"Transfer-Encoding: chunked\r\n",
            b"Connection: close\r\n",
            b"\r\n",
            chunk_size + b"\r\n" + payload + b"\r\n",
            b"0\r\n\r\n",
        ]
    )

    with socket.create_connection((host, port), timeout=5) as sock:
        sock.sendall(req)
        sock.shutdown(socket.SHUT_WR)
        response = b""
        while b"\r\n" not in response:
            data = sock.recv(4096)
            if not data:
                break
            response += data

    status_line = response.split(b"\r\n", 1)[0].decode("ascii", errors="replace")
    parts = status_line.split(" ")
    if len(parts) < 2 or not parts[1].isdigit():
        _fail(f"unexpected HTTP status line: {status_line!r}")
    return int(parts[1])


def _validate_cartridge_info(info: object) -> tuple[list[str], str, str]:
    if not isinstance(info, dict):
        _fail("/cartridgeinfo should return a JSON object")

    for key in ("present", "mounted", "isRetroPie", "systems", "emptySystems", "busy"):
        if key not in info:
            _fail(f"/cartridgeinfo missing key {key!r}")

    systems = info["systems"]
    if not isinstance(systems, list):
        _fail("/cartridgeinfo systems should be a JSON array")

    system_counts: dict[str, int] = {}
    for entry in systems:
        if not isinstance(entry, dict):
            _fail("/cartridgeinfo systems entries should be JSON objects")
        system_name = entry.get("system")
        file_count = entry.get("filecount")
        if not isinstance(system_name, str):
            _fail("/cartridgeinfo systems entries need a string 'system'")
        if not isinstance(file_count, int):
            _fail("/cartridgeinfo systems entries need an integer 'filecount'")
        system_counts[system_name] = file_count

    expected_counts = {"nes": 1, "snes": 1}
    if system_counts != expected_counts:
        _fail(f"/cartridgeinfo systems mismatch: expected {expected_counts}, got {system_counts}")

    empty_systems = info["emptySystems"]
    if not isinstance(empty_systems, list) or not all(isinstance(x, str) for x in empty_systems):
        _fail("/cartridgeinfo emptySystems should be a JSON string array")
    if empty_systems != ["pc"]:
        _fail(f"/cartridgeinfo emptySystems mismatch: expected ['pc'], got {empty_systems}")

    all_systems = sorted([*system_counts.keys(), *empty_systems])
    return all_systems, "nes", "pc"


def main() -> None:
    base = os.environ.get("KEYMAKER_API_BASE", "http://127.0.0.1:8080").strip()
    if not base:
        _fail("KEYMAKER_API_BASE is empty")

    _wait_up(base)

    # /cartridgeinfo
    status, _, body = _request("GET", base, "/api/v1/cartridgeinfo")
    if status != 200:
        _fail(f"GET /api/v1/cartridgeinfo: expected 200, got {status}")
    info = json.loads(body.decode("utf-8"))
    expected_systems, non_empty_system, empty_system = _validate_cartridge_info(info)

    # /retropie
    status, _, body = _request("GET", base, "/api/v1/retropie")
    if status != 200:
        _fail(f"GET /api/v1/retropie: expected 200, got {status}")
    systems = json.loads(body.decode("utf-8"))
    if not isinstance(systems, list) or not all(isinstance(x, str) for x in systems):
        _fail("/retropie should return a JSON string array")
    if sorted(systems) != expected_systems:
        _fail(f"/retropie systems mismatch: expected {expected_systems}, got {systems}")

    # /retropie/{empty_system}
    status, _, body = _request("GET", base, f"/api/v1/retropie/{empty_system}")
    if status != 200:
        _fail(f"GET /api/v1/retropie/{empty_system}: expected 200, got {status}")
    empty_games = json.loads(body.decode("utf-8"))
    if empty_games != []:
        _fail(f"/retropie/{empty_system} should return an empty list")

    # /retropie/{system}
    status, _, body = _request("GET", base, f"/api/v1/retropie/{non_empty_system}")
    if status != 200:
        _fail(f"GET /api/v1/retropie/{non_empty_system}: expected 200, got {status}")
    games = json.loads(body.decode("utf-8"))
    if not isinstance(games, list) or not all(isinstance(x, str) for x in games):
        _fail(f"/retropie/{non_empty_system} should return a JSON string array")
    if not games:
        _fail(f"/retropie/{non_empty_system} returned empty game list")

    game = games[0]

    # download should return bytes and Content-Disposition
    status, headers, _ = _request("GET", base, f"/api/v1/retropie/{non_empty_system}/{game}")
    if status != 200:
        _fail(f"GET /api/v1/retropie/{non_empty_system}/{game}: expected 200, got {status}")
    header_keys = {k.lower() for k in headers.keys()}
    if "content-disposition" not in header_keys:
        _fail("download response missing Content-Disposition header")

    # upload + delete (use a unique name)
    upload_name = f"validate-upload-{int(time.time())}.bin"
    payload = b"hello-keymaker"
    status, _, body = _request(
        "POST",
        base,
        f"/api/v1/retropie/{non_empty_system}/{upload_name}",
        body=payload,
        headers={"Content-Type": "application/octet-stream", "Content-Length": str(len(payload))},
    )
    if status != 200:
        _fail(f"POST upload: expected 200, got {status}")
    upload_resp = json.loads(body.decode("utf-8"))
    if upload_resp.get("ok") is not True:
        _fail("POST upload: expected {ok:true}")

    status, _, body = _request("DELETE", base, f"/api/v1/retropie/{non_empty_system}/{upload_name}")
    if status != 200:
        _fail(f"DELETE game: expected 200, got {status}")
    delete_resp = json.loads(body.decode("utf-8"))
    if delete_resp.get("ok") is not True:
        _fail("DELETE game: expected {ok:true}")

    # flash: should accept with 202 when Content-Length is present
    flash_payload = b"x" * 32
    status, _, body = _request(
        "POST",
        base,
        "/api/v1/flash",
        body=flash_payload,
        headers={"Content-Type": "application/octet-stream", "Content-Length": str(len(flash_payload))},
    )
    if status != 202:
        _fail(f"POST /flash: expected 202, got {status}")
    flash_resp = json.loads(body.decode("utf-8"))
    if flash_resp.get("ok") is not True:
        _fail("POST /flash: expected {ok:true}")

    # flash without Content-Length: should return 411
    status = _request_raw_chunked_status(base, "/api/v1/flash", b"abc")
    if status != 411:
        _fail(f"POST /flash chunked: expected 411, got {status}")

    print("ok")


if __name__ == "__main__":
    main()
