# Experimental prototype

This is a Go 1.23 prototype with an **experimental** command in `cmd/bridge`. It is not production-ready.

## Reproduction

From this directory:

```sh
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

Local verification on 2026-11-15 with Go 1.23.2 linux/amd64: all three commands passed.

Tests cover 12 concurrent bidirectional 48 KiB echo streams, fallback from a refused TCP backend to an available one, and rejection of an unauthenticated request.

## Known blockers

Do not deploy. No robust half-close or early-backend-EOF cancellation test, per-identity service authorization, resource limits, timeouts, metrics, configuration loader, real VPN profile tests or route benchmarks. TLS test credentials are generated ephemerally inside the tests.

## Experimental CLI

From `prototype/`, build with `go build ./cmd/bridge` and run `./bridge -config config.json`. Both modes require `listen`, `cert_file`, `key_file`, and `ca_file`. Never commit real private keys or configuration containing secrets.

Server example (replace the fingerprint and certificate paths):

```json
{
  "mode": "server", "listen": "127.0.0.1:8443",
  "cert_file": "/secure/server.pem", "key_file": "/secure/server.key",
  "ca_file": "/secure/client-ca.pem", "max_streams": 32,
  "routes": {"vpn": [{"Name": "echo", "Address": "127.0.0.1:9000"}]},
  "authorized": {"REPLACE_WITH_64_HEX_SHA256_CERT_FINGERPRINT": ["vpn"]}
}
```

Client example:

```json
{
  "mode": "client", "listen": "127.0.0.1:9001",
  "cert_file": "/secure/client.pem", "key_file": "/secure/client.key",
  "ca_file": "/secure/server-ca.pem", "server_name": "bridge.example.org",
  "remote_url": "https://bridge.example.org:8443/v1/stream",
  "service": "vpn", "max_local_connections": 32
}
```

The server certificate fingerprint is the SHA-256 of the client leaf certificate DER (not the PEM text). Use distinct server and client certificates and trusted CA material. The example server binds to loopback only; external access requires an explicitly controlled firewall and further security testing.

**Not yet validated for deployment:** full exact-commit CI, multi-VPS connectivity, signal handling with real streams, throughput, resource exhaustion and backend disconnect tests. Do not deploy to production.
