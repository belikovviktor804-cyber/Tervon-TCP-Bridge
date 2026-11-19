# Security policy — Tervon TCP Bridge v1.0.1

Tervon TCP Bridge provides a bidirectional TCP stream bridge over HTTP/2 with TLS 1.3 and mutual certificate authentication (mTLS). The administrator configures the permitted service names and fixed backend allowlists. The bridge does not decrypt or interpret the application protocol carried inside a TCP stream.

**Supported release:** v1.0.1 for Linux amd64. Use an isolated first installation, review the configuration before activation, and do not publish private PKI materials, production routes, tokens or traffic captures.

Please report vulnerabilities privately through [GitHub private vulnerability reporting](https://github.com/tervon-software/Tervon-TCP-Bridge/security/advisories/new) when enabled. If that feature is unavailable, contact the maintainer through a nonpublic channel; do not place exploitable details or credentials in public issues.

Security controls include TLS 1.3, mTLS, per-certificate service authorization, server-configured backend routing, bounded connection setup, cancellation and non-payload aggregate metrics. **v1.0.1 disables fixed application-level session quotas by default** (`max_local_connections=0`, `max_streams=0`). This does not remove operating-system resource limits and is not a guarantee against denial-of-service or arbitrary network filtering.

The installer is for first deployment only; it refuses to overwrite a running installation and does not automatically activate services. For supported public setup instructions see [v1.0.1 installation](docs/INSTALL-v1.0.1.ru.md).
