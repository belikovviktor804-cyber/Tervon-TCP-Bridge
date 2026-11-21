package com.tervonvpndpi.app

/**
 * Routing decision only. Never changes VPN configuration or starts Bridge.
 * The caller must establish the authorized mTLS Bridge, check local readiness,
 * and identify the active VLESS/TCP destination before applying this plan.
 */
internal object BridgeVlessRoutePlan {
    data class Endpoint(val host: String, val port: Int)
    data class Route(val remote: Endpoint, val local: Endpoint)

    sealed class Decision {
        object Direct : Decision()
        data class ThroughBridge(val route: Route) : Decision()
        data class Blocked(val reason: Reason) : Decision()
    }

    enum class Reason {
        INVALID_PROFILE, UNSUPPORTED_TRANSPORT, BRIDGE_NOT_READY,
        INVALID_LOCAL_LISTENER, WRONG_SERVER
    }

    fun decide(
        enabled: Boolean,
        protocol: String,
        transport: String,
        activeServer: Endpoint?,
        authorizedServer: Endpoint?,
        localPort: Int,
        bridgeReady: Boolean
    ): Decision {
        if (!enabled) return Decision.Direct
        if (!protocol.equals("vless", ignoreCase = true) ||
            !transport.equals("tcp", ignoreCase = true)) {
            return Decision.Blocked(Reason.UNSUPPORTED_TRANSPORT)
        }
        if (activeServer == null || !validRemote(activeServer)) {
            return Decision.Blocked(Reason.INVALID_PROFILE)
        }
        if (authorizedServer == null || !validRemote(authorizedServer) ||
            !activeServer.host.trimEnd('.').equals(authorizedServer.host.trimEnd('.'), ignoreCase = true) ||
            activeServer.port != authorizedServer.port) {
            return Decision.Blocked(Reason.WRONG_SERVER)
        }
        if (localPort !in 1024..65535) {
            return Decision.Blocked(Reason.INVALID_LOCAL_LISTENER)
        }
        if (!bridgeReady) return Decision.Blocked(Reason.BRIDGE_NOT_READY)
        return Decision.ThroughBridge(Route(activeServer, Endpoint("127.0.0.1", localPort)))
    }

    private fun validRemote(endpoint: Endpoint): Boolean {
        val host = endpoint.host.trimEnd('.')
        return host.isNotBlank() && host.length <= 253 &&
            host.none { it.isWhitespace() || it in "/\\@?#" } &&
            endpoint.port in 1..65535
    }
}
