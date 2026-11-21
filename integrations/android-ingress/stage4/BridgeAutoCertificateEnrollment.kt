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

    @Synchronized
    fun onBypassEnabled(enabled: Boolean, trusted: TrustedBridge?): Outcome {
        if (!enabled) return Outcome.Disabled
        if (trusted == null) return Outcome.NoBridge
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
