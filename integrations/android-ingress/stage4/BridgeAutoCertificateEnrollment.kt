package com.tervonvpndpi.app

/**
 * Runs only after the user switches "Обход DPI" ON.
 * The current subscription must first supply an authenticated Bridge identity
 * and its authorized certificate-issuance transport. No domain or port
 * is compiled into the application.
 *
 * This coordinator does not invent a discovery endpoint or a second login.
 */
internal class BridgeAutoCertificateEnrollment(
    private val identity: DeviceIdentity,
    private val certificates: CertificateStorage,
    private val transport: EnrollmentTransport,
) {
    data class TrustedBridge(val serverId: String, val clientCaPem: String) {
        init {
            require(serverId.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,127}")))
            require(clientCaPem.isNotBlank())
        }
    }

    data class Issued(val certificatePem: String)

    sealed class Outcome {
        object Disabled : Outcome()
        object NoBridge : Outcome()
        data class Reused(val serverId: String) : Outcome()
        data class Enrolled(val serverId: String) : Outcome()
    }

    interface DeviceIdentity {
        fun spkiDer(): ByteArray
        fun csrPem(): String
    }

    interface CertificateStorage {
        fun hasValid(serverId: String, clientCaPem: String, spkiDer: ByteArray): Boolean
        fun install(serverId: String, certificatePem: String, clientCaPem: String, spkiDer: ByteArray)
    }

    interface EnrollmentTransport {
        /** Only a subscription-authenticated server endpoint may issue. */
        fun issue(trusted: TrustedBridge, csrPem: String): Issued
    }

    interface SubscriptionBridgeDiscovery {
        /**
         * Must use the already authenticated subscription response and verify
         * whether this subscription server has configured Bridge support.
         * No DNS/SNI-based trust and no new registration credentials.
         */
        fun discover(activeSubscription: String): TrustedBridge?
    }

    /**
     * The ONLY public entry point: discovery and CSR happen exclusively after
     * the user enables the bypass toggle, never on subscription refresh or OFF.
     * The caller supplies the actual selected subscription, not a hardcoded host.
     */
    @Synchronized
    fun onBypassToggled(
        enabled: Boolean,
        activeSubscription: String?,
        discovery: SubscriptionBridgeDiscovery,
    ): Outcome {
        if (!enabled) return Outcome.Disabled
        if (activeSubscription.isNullOrBlank()) return Outcome.NoBridge
        val trusted = discovery.discover(activeSubscription) ?: return Outcome.NoBridge
        val publicKey = identity.spkiDer()
        if (certificates.hasValid(trusted.serverId, trusted.clientCaPem, publicKey)) {
            return Outcome.Reused(trusted.serverId)
        }
        val csr = identity.csrPem()
        val issued = transport.issue(trusted, csr)
        // Existing store checks issuer, clientAuth, expiry and exact device key.
        certificates.install(trusted.serverId, issued.certificatePem, trusted.clientCaPem, publicKey)
        return Outcome.Enrolled(trusted.serverId)
    }
}
