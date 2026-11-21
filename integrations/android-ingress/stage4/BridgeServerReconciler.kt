package com.tervonvpndpi.app

import java.util.Locale

/**
 * Pure, offline M6 policy. No DNS, network, file I/O, side effects or enrollment.
 *
 * The caller must supply:
 *  - the real selected profile's destination HOST + PORT (not camouflage SNI);
 *  - server registrations from an authenticated, trusted catalog;
 *  - saved per-server certificates from app-private storage;
 *  - the actual current Bridge server ID (when running).
 *
 * This class never enables Bridge on its own. It does not check whether VLESS
 * credentials work, and does not equate a connection error with an unknown host.
 */
internal object BridgeServerReconciler {
    data class Endpoint(val host: String, val port: Int)

    data class Server(
        val id: String,
        val endpoints: Set<Endpoint>,
        val tlsName: String,
    )

    data class Certificate(
        val serverId: String,
        val notAfterEpochMillis: Long,
    )

    enum class Action {
        /** The operator has switched Bridge OFF; no Bridge work is permitted. */
        IDLE_OFF,
        /** Existing Bridge server/certificate remains selected. */
        KEEP_CURRENT,
        /** Switch only to a previously issued valid certificate. */
        SELECT_SAVED,
        /** No valid certificate: request automatic authorized enrollment. */
        REQUEST_AUTO_ENROLLMENT,
        /** No trusted Bridge mapping for the selected VPN destination. */
        TURN_OFF_UNKNOWN,
        /** Missing/invalid selected VPN destination; fail closed. */
        TURN_OFF_INVALID_PROFILE,
        /** Two trusted records claim the same endpoint; fail closed. */
        TURN_OFF_AMBIGUOUS,
    }

    data class Decision(
        val action: Action,
        val serverId: String? = null,
        val certificate: Certificate? = null,
    )

    private fun canonicalHost(value: String): String? {
        // No arbitrary URI, SNI or opaque identifier should resolve as a host.
        val host = value.trim().removeSuffix(".").lowercase(Locale.ROOT)
        if (host.isEmpty() || host.length > 253 ||
            host.any { it.isWhitespace() || it in "/\\@?#:" }) return null
        if (host.startsWith(".") || host.contains("..")) return null
        return host
    }

    private fun canonical(endpoint: Endpoint): Endpoint? {
        val name = canonicalHost(endpoint.host) ?: return null
        if (endpoint.port !in 1..65535) return null
        return Endpoint(name, endpoint.port)
    }

    fun decide(
        bridgeEnabled: Boolean,
        selectedDestination: Endpoint?,
        trustedServers: Collection<Server>,
        savedCertificates: Collection<Certificate>,
        currentBridgeServerId: String?,
        nowEpochMillis: Long,
    ): Decision {
        // This MUST precede any selected-profile or catalog processing.
        if (!bridgeEnabled) return Decision(Action.IDLE_OFF)

        val destination = selectedDestination?.let(::canonical)
            ?: return Decision(Action.TURN_OFF_INVALID_PROFILE)

        // Trusted records are already authenticated by the caller. A collision
        // is a configuration error, never a reason to guess or use first match.
        val matches = trustedServers.filter { server ->
            server.id.isNotBlank() && server.tlsName.isNotBlank() &&
                server.endpoints.any { canonical(it) == destination }
        }
        if (matches.isEmpty()) return Decision(Action.TURN_OFF_UNKNOWN)
        if (matches.size != 1) return Decision(Action.TURN_OFF_AMBIGUOUS)

        val server = matches.single()
        val valid = savedCertificates.filter {
            it.serverId == server.id && it.notAfterEpochMillis > nowEpochMillis
        }.maxByOrNull { it.notAfterEpochMillis }

        if (valid == null) {
            return Decision(Action.REQUEST_AUTO_ENROLLMENT, server.id)
        }
        return if (currentBridgeServerId == server.id) {
            Decision(Action.KEEP_CURRENT, server.id, valid)
        } else {
            Decision(Action.SELECT_SAVED, server.id, valid)
        }
    }
}
