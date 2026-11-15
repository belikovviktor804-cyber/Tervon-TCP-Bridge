# Security policy

This project is experimental and must not be deployed to production.

Please report vulnerabilities privately through GitHub's private vulnerability reporting when available, rather than publishing exploit details. Never submit real credentials, private keys, production IP inventories or user traffic captures.

Security gates before release: strict mTLS validation; per-client service authorization; static backend allowlist; resource quotas; bounded cancellation; idle and connection deadlines; no payload logging.
