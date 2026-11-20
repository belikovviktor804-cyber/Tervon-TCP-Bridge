package com.tervonvpndpi.app

import android.content.Context
import android.util.AtomicFile
import android.util.Base64
import java.io.ByteArrayInputStream
import java.io.File
import java.security.MessageDigest
import java.security.cert.CertificateExpiredException
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import java.util.Date

/**
 * M6 candidate: isolated, app-private per-Bridge-server client certificate store.
 *
 * Server ID and CA must come from a verified/trusted catalog, never from an
 * arbitrary VPN URL or user-controlled SNI. There is no network or enrollment.
 * A valid certificate is useful only when a user explicitly enrolls it.
 *
 * This class never handles or copies the private device key.
 */
internal class BridgeServerCertificateStore(context: Context) {
    private val app = context.applicationContext
    private val directory = File(app.noBackupFilesDir, "bridge-server-client-certificates-v1")
    private val factory = CertificateFactory.getInstance("X.509")

    data class StoredCertificate(
        val serverId: String,
        val certificatePem: String,
        val notAfterEpochMillis: Long,
    )

    private fun requireId(serverId: String) {
        require(serverId.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,127}"))) {
            "invalid trusted Bridge server ID"
        }
    }

    private fun storage(serverId: String): AtomicFile {
        requireId(serverId)
        val digest = MessageDigest.getInstance("SHA-256")
            .digest(serverId.toByteArray(Charsets.UTF_8))
        val fileName = digest.joinToString("") { "%02x".format(it.toInt() and 0xff) } + ".pem"
        return AtomicFile(File(directory, fileName))
    }

    private fun parsePem(pem: String): X509Certificate {
        require(pem.length in 100..32768) { "invalid X.509 PEM size" }
        val pattern = Regex(
            "^\\s*-----BEGIN CERTIFICATE-----\\s*([A-Za-z0-9+/=\\s]+)" +
                "-----END CERTIFICATE-----\\s*$"
        )
        val match = pattern.matchEntire(pem) ?: error("expected a single X.509 PEM certificate")
        val der = Base64.decode(match.groupValues[1], Base64.DEFAULT)
        return ByteArrayInputStream(der).use {
            factory.generateCertificate(it) as X509Certificate
        }
    }

    private fun verifiedCertificate(
        pem: String,
        caPem: String,
        expectedPublicKey: ByteArray,
        nowEpochMillis: Long,
    ): X509Certificate {
        val client = parsePem(pem)
        val ca = parsePem(caPem)
        val now = Date(nowEpochMillis)
        client.checkValidity(now)
        ca.checkValidity(now)

        require(ca.basicConstraints >= 0) { "Bridge authority is not a CA" }
        require(client.basicConstraints < 0) { "client certificate must not be a CA" }
        require(client.issuerX500Principal == ca.subjectX500Principal) {
            "client certificate issuer differs from pinned authority"
        }
        client.verify(ca.publicKey)
        require(client.publicKey.encoded.contentEquals(expectedPublicKey)) {
            "certificate does not belong to this Android device"
        }
        val extendedUses = client.extendedKeyUsage
        require(extendedUses != null && "1.3.6.1.5.5.7.3.2" in extendedUses) {
            "client certificate is not permitted for TLS client authentication"
        }
        return client
    }

    /**
     * Persist ONLY after explicit user enrollment and trusted server metadata
     * validation. AtomicFile avoids partial replacement on interruption.
     */
    @Synchronized
    fun install(
        trustedServerId: String,
        clientCertificatePem: String,
        pinnedCaPem: String,
        devicePublicKeySpkiDer: ByteArray,
        nowEpochMillis: Long = System.currentTimeMillis(),
    ): StoredCertificate {
        requireId(trustedServerId)
        val cert = verifiedCertificate(
            clientCertificatePem, pinnedCaPem, devicePublicKeySpkiDer, nowEpochMillis
        )
        check(directory.isDirectory || directory.mkdirs()) {
            "cannot create private Bridge certificate directory"
        }
        val target = storage(trustedServerId)
        val output = target.startWrite()
        try {
            output.write(clientCertificatePem.toByteArray(Charsets.UTF_8))
            target.finishWrite(output)
        } catch (e: Exception) {
            target.failWrite(output)
            throw e
        }
        return StoredCertificate(trustedServerId, clientCertificatePem, cert.notAfter.time)
    }

    /** No DNS or network access; expiry, device key and pinned issuer checked on read. */
    @Synchronized
    fun findValid(
        trustedServerId: String,
        pinnedCaPem: String,
        devicePublicKeySpkiDer: ByteArray,
        nowEpochMillis: Long = System.currentTimeMillis(),
    ): StoredCertificate? {
        val target = storage(trustedServerId)
        if (!target.baseFile.isFile) return null
        val bytes = target.openRead().use { stream ->
            val data = stream.readBytes()
            require(data.size in 100..32768) { "invalid stored certificate size" }
            data
        }
        val pem = String(bytes, Charsets.UTF_8)
        val cert = try {
            verifiedCertificate(pem, pinnedCaPem, devicePublicKeySpkiDer, nowEpochMillis)
        } catch (_: CertificateExpiredException) {
            // Preserve the old PEM for diagnostics/explicit renewal; never auto-enroll.
            return null
        }
        return StoredCertificate(trustedServerId, pem, cert.notAfter.time)
    }
}
