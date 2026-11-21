#!/usr/bin/env python3
"""Prepare a PRIVATE M6 XHTTP route candidate from actual running Xray sources.

NEVER edits active Xray, Bridge config, Docker, firewall, or client subscriptions.
A candidate needs a separately approved staging/live cutover and tests.
"""
import argparse
import json
import os
import pathlib
import sys


def candidate(xray: dict, bridge: dict) -> dict:
    if bridge.get("mode") != "server" or not isinstance(bridge.get("routes"), dict):
        raise ValueError("Bridge server configuration invalid")
    routes = bridge["routes"]
    tcp = routes.get("vpn")
    if not isinstance(tcp, list) or len(tcp) != 1:
        raise ValueError("existing VPN TCP backend must be unique")
    current = tcp[0]
    if not isinstance(current, dict):
        raise ValueError("existing Bridge backend invalid")
    address = current.get("Address", current.get("address", ""))
    if not isinstance(address, str) or ":" not in address:
        raise ValueError("existing backend address invalid")
    host, tcpport = address.rsplit(":", 1)
    if not host or not tcpport.isdecimal() or int(tcpport) not in range(1, 65536):
        raise ValueError("invalid existing Bridge host or port")
    options = []
    for entry in xray.get("inbounds", []):
        if entry.get("protocol") != "vless":
            continue
        stream = entry.get("streamSettings") or {}
        if str(stream.get("network", "")).lower() not in ("xhttp", "splithttp"):
            continue
        listener = str(entry.get("listen", "0.0.0.0"))
        if listener in ("127.0.0.1", "::1", "localhost"):
            raise ValueError("XHTTP inbound is loopback-only and unreachable by the Bridge container")
        port = entry.get("port")
        if isinstance(port, int) and 1 <= port <= 65535:
            options.append(port)
    if len(options) != 1:
        raise ValueError("expected exactly one reachable VLESS XHTTP inbound")
    if "vpn-xhttp" in routes:
        raise ValueError("XHTTP Bridge route already configured; do not overwrite")
    output = json.loads(json.dumps(bridge))
    # Reuse the EXACT verified Docker host gateway of the working TCP backend.
    output["routes"]["vpn-xhttp"] = [{"Name": "xhttp", "Address": f"{host}:{options[0]}"}]
    acl = output.get("authorized")
    if not isinstance(acl, dict):
        raise ValueError("existing Bridge ACL missing")
    for fingerprint, allowed in acl.items():
        if not isinstance(allowed, list):
            raise ValueError("invalid existing ACL entry")
        if "vpn" in allowed and "vpn-xhttp" not in allowed:
            allowed.append("vpn-xhttp")
    return output


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--xray-config", type=pathlib.Path, required=True)
    parser.add_argument("--bridge-config", type=pathlib.Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--stage", type=pathlib.Path)
    args = parser.parse_args()
    try:
        # Source-only: no private config is logged or committed.
        xray = json.loads(args.xray_config.read_text())
        bridge = json.loads(args.bridge_config.read_text())
        result = candidate(xray, bridge)
        port = result["routes"]["vpn-xhttp"][0]["Address"].rsplit(":", 1)[1]
        print("M6_XHTTP_CANDIDATE=VALID")
        print("XHTTP_BACKEND_PORT=" + port)
        print("LIVE_SERVICES=UNCHANGED")
        if args.check:
            return 0
        target = args.stage
        if not target.is_absolute() or target.is_symlink():
            raise ValueError("stage output must be a new absolute regular file")
        content = (json.dumps(result, indent=2, sort_keys=True) + "\n").encode()
        fd = os.open(str(target), os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        try:
            with os.fdopen(fd, "wb") as output:
                output.write(content)
                output.flush()
                os.fsync(output.fileno())
        except Exception:
            target.unlink(missing_ok=True)
            raise
        print("M6_XHTTP_PRIVATE_CANDIDATE=STAGED")
        print("LIVE_CONFIG_CHANGED=NO")
        return 0
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as e:
        # Do not print raw JSON, exception paths or credentials.
        print("M6_XHTTP_CANDIDATE=BLOCKED_" + type(e).__name__)
        return 1


if __name__ == "__main__":
    sys.exit(main())
