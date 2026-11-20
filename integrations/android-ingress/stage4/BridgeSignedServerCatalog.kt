package com.tervonvpndpi.app

import android.content.Context
import android.util.AtomicFile
import android.util.Base64
import org.bouncycastle.crypto.params.Ed25519PublicKeyParameters
import org.bouncycastle.crypto.signers.Ed25519Signer
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.net.URI
import java.nio.charset.StandardCharsets
import java.security.MessageDigest

/**
 * M6 signed catalog verifier + offline cache. No DNS, TLS, registration or VPN actions.
 *
 * trustedPublicKeys MUST come from a fixed trusted app/build configuration, never
 * from the fetched catalog or arbitrary subscription metadata. Do not instantiate
 * this class with untrusted runtime keys.
 *
 * The signature covers: UTF-8("Tervon-Bridge-Catalog-v1\0") || raw payload bytes.
 * The signed payload is the exact Base64-decoded UTF-8 JSON; no JSON re-serialization.
 */
internal class BridgeSignedServerCatalog(
    context: Context,
    trustedPublicKeys: Map<String, ByteArray>,
) {
    private val file = AtomicFile(
        File(context.applicationContext.noBackupFilesDir, "bridge-signed-catalog-v1.json")
    )
    private val trustedKeys = trustedPublicKeys.mapValues { (_, value) -> value.clone() }
    private val prefix = "Tervon-Bridge-Catalog-v1\u0000".toByteArray(StandardCharsets.UTF_8)

    data class Server(
        val id: String,
        val endpoints: Set<BridgeServerReconciler.Endpoint>,
        val tlsName: String,
        val bridgeUrl: String,
        val enrollmentUrl: String,
        val serverCaPem: String,
    )

    data class Catalog(
        val serial: Long,
        val issuedAtEpochSeconds: Long,
        val expiresAtEpochSeconds: Long,
        val servers: List<Server>,
    ) {
        fun forReconciler(): List<BridgeServerReconciler.Server> =
            servers.map { BridgeServerReconciler.Server(it.id, it.endpoints, it.tlsName) }
    }

    private data class Parsed(val catalog: Catalog, val envelopeBytes: ByteArray)

    private fun decode64(value: String, maxBytes: Int): ByteArray {
        require(value.length <= maxBytes * 2 + 8) { "catalog field oversized" }
        val bytes = Base64.decode(value, Base64.NO_WRAP)
        require(bytes.size in 1..maxBytes) { "invalid catalog field length" }
        require(Base64.encodeToString(bytes, Base64.NO_WRAP) == value) {
            "noncanonical catalog Base64"
        }
        return bytes
    }

    private fun canonicalHost(value: String): String {
        val host = value.trim().lowercase(java.util.Locale.ROOT).removeSuffix(".")
        require(host.isNotEmpty() && host.length <= 253 &&
            !host.startsWith(".") && !host.contains("..") &&
            host.all { it.isLetterOrDigit() || it == '-' || it == '.' }) {
            "invalid catalog host"
        }
        return host
    }

    private fun httpsUrl(value: String) {
        val url = URI(value)
        require(url.scheme == "https" && url.host != null && url.rawUserInfo == null &&
            url.rawQuery == null && url.rawFragment == null && url.port in -1..65535 &&
            url.port != 0 && url.host == canonicalHost(url.host)) {
            "invalid catalog HTTPS URL"
        }
    }

    private fun exactFields(obj: JSONObject, required: Set<String>) {
        val keys = obj.keys().asSequence().toSet()
        require(keys == required) { "unrecognized catalog fields" }
    }

    private fun verify(bytes: ByteArray, nowSeconds: Long, enforceTime: Boolean): Parsed {
        require(bytes.size in 30..131072) { "invalid envelope size" }
        val envelope = JSONObject(String(bytes, StandardCharsets.UTF_8))
        exactFields(envelope, setOf("format", "key_id", "payload_b64", "signature_b64"))
        require(envelope.getString("format") == "tervon.bridge.catalog.envelope.v1")
        val keyId = envelope.getString("key_id")
        require(keyId.matches(Regex("[A-Za-z0-9_-]{1,64}"))) { "invalid catalog key id" }
        val publicKey = trustedKeys[keyId] ?: error("untrusted catalog signing key")
        require(publicKey.size == 32) { "invalid pinned public key size" }
        val payload = decode64(envelope.getString("payload_b64"), 65536)
        val signature = decode64(envelope.getString("signature_b64"), 64)
        require(signature.size == 64) { "invalid catalog signature size" }

        val verifier = Ed25519Signer()
        verifier.init(false, Ed25519PublicKeyParameters(publicKey, 0))
        verifier.update(prefix, 0, prefix.size)
        verifier.update(payload, 0, payload.size)
        require(verifier.verifySignature(signature)) { "catalog signature mismatch" }

        // Never consume untrusted catalog fields until the Ed25519 check has passed.
        val root = JSONObject(String(payload, StandardCharsets.UTF_8))
        exactFields(root, setOf("version", "serial", "issued_at", "expires_at", "servers"))
        require(root.getInt("version") == 1) { "unsupported catalog version" }
        val serial = root.getLong("serial")
        val issued = root.getLong("issued_at")
        val expires = root.getLong("expires_at")
        require(serial > 0 && issued > 0 && expires > issued &&
            expires - issued <= 30L * 86400L) { "invalid catalog serial or validity" }
        if (enforceTime) {
            require(issued <= nowSeconds + 300 && nowSeconds < expires) {
                "catalog is not currently valid"
            }
        }

        val list = root.getJSONArray("servers")
        require(list.length() in 1..256) { "invalid catalog server count" }
        val ids = HashSet<String>()
        val assignedEndpoints = HashSet<BridgeServerReconciler.Endpoint>()
        val servers = ArrayList<Server>(list.length())
        for (i in 0 until list.length()) {
            val server = list.getJSONObject(i)
            exactFields(server, setOf(
                "id", "endpoints", "tls_name", "bridge_url", "enrollment_url", "server_ca_pem"
            ))
            val id = server.getString("id")
            require(id.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,127}")) &&
                ids.add(id)) { "duplicate or invalid Bridge server id" }
            val tlsName = canonicalHost(server.getString("tls_name"))
            val bridgeUrl = server.getString("bridge_url")
            val enrollmentUrl = server.getString("enrollment_url")
            httpsUrl(bridgeUrl)
            httpsUrl(enrollmentUrl)
            require(URI(bridgeUrl).host == tlsName) {
                "Bridge HTTPS endpoint does not match signed TLS name"
            }
            val caPem = server.getString("server_ca_pem")
            require(caPem.length in 128..16384 &&
                caPem.startsWith("-----BEGIN CERTIFICATE-----") &&
                caPem.trimEnd().endsWith("-----END CERTIFICATE-----")) {
                "invalid catalog authority PEM"
            }

            val entries = server.getJSONArray("endpoints")
            require(entries.length() in 1..128) { "invalid server endpoint count" }
            val endpoints = LinkedHashSet<BridgeServerReconciler.Endpoint>()
            for (j in 0 until entries.length()) {
                val e = entries.getJSONObject(j)
                exactFields(e, setOf("host", "port"))
                val ep = BridgeServerReconciler.Endpoint(
                    canonicalHost(e.getString("host")), e.getInt("port")
                )
                require(ep.port in 1..65535 && endpoints.add(ep) &&
                    assignedEndpoints.add(ep)) { "duplicate or invalid catalog endpoint" }
            }
            servers.add(Server(id, endpoints, tlsName, bridgeUrl, enrollmentUrl, caPem))
        }
        return Parsed(Catalog(serial, issued, expires, servers), bytes)
    }

    /** Offline only. An expired catalog is never returned as trusted. */
    @Synchronized
    fun current(nowEpochSeconds: Long = System.currentTimeMillis() / 1000): Catalog? {
        if (!file.baseFile.isFile) return null
        val bytes = file.openRead().use { input ->
            val b = input.readBytes()
            require(b.size <= 131072) { "cached catalog oversized" }
            b
        }
        return verify(bytes, nowEpochSeconds, true).catalog
    }

    /**
     * Accept only a signed, unexpired update with strictly increasing serial.
     * The exact same signed envelope is idempotent; equal-serial replacements
     * and rollback to an older catalog are denied. Invalid updates do not
     * modify the previous cached catalog.
     */
    @Synchronized
    fun acceptUpdate(
        envelopeBytes: ByteArray,
        nowEpochSeconds: Long = System.currentTimeMillis() / 1000,
    ): Catalog {
        val next = verify(envelopeBytes, nowEpochSeconds, true)
        if (file.baseFile.isFile) {
            val previousBytes = file.openRead().use { input ->
                val b = input.readBytes()
                require(b.size <= 131072) { "cached catalog oversized" }
                b
            }
            val previous = verify(previousBytes, nowEpochSeconds, false)
            require(next.catalog.serial > previous.catalog.serial ||
                (next.catalog.serial == previous.catalog.serial &&
                 MessageDigest.isEqual(envelopeBytes, previousBytes))) {
                "catalog serial rollback or conflicting catalog"
            }
            if (MessageDigest.isEqual(envelopeBytes, previousBytes)) return next.catalog
        }
        val output = file.startWrite()
        try {
            output.write(envelopeBytes)
            file.finishWrite(output)
        } catch (e: Exception) {
            file.failWrite(output)
            throw e
        }
        return next.catalog
    }
}
