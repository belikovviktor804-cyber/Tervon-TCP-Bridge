# Experimental prototype

This is a Go 1.23 library, not a deployable binary.

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
