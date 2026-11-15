# Tervon TCP Bridge

**Status: experimental architecture and prototype; not production-ready.**

Tervon TCP Bridge is a proposed open-source TCP byte-stream relay using real HTTP/2 over TLS 1.3.

```
TCP client -> local Encapsulator -> HTTPS/HTTP2 -> remote Decapsulator -> allowlisted TCP backend
```

## Goals

- Secure, high-throughput bidirectional TCP relay.
- Minimize distinguishable observable traffic characteristics; **no guarantee of invisibility to DPI**.
- Measure throughput, connection setup, resource usage and reliability on specified routes.
- Keep bridge independent of Guard, Processor and Tervon Federation.

## Scope v0.1

TCP only. VPN protocols are opaque to the bridge. VLESS, Trojan, Shadowsocks and OpenVPN configurations require individual compatibility tests. No native UDP, WireGuard, Hysteria2, CONNECT-UDP, CDN, uTLS, Cloak or Meek in v0.1.

The remote server selects from a fixed allowlist of backends at connection establishment. Clients cannot supply arbitrary destination addresses. No live migration of established TCP sessions.

## Current status

This repository starts with the design charter. The experimental Go prototype is **not yet imported or validated in this repository**. Do not deploy to production.

See [Project charter](docs/PROJECT_CHARTER.md) and [development roadmap](docs/ROADMAP.md).

## Licensing

An explicit open-source license is pending confirmation. Until a license is added, public availability does not grant reuse rights.
