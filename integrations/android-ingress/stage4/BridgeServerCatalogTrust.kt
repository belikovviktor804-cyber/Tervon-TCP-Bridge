package com.tervonvpndpi.app

import android.content.Context
import android.util.AtomicFile
import android.util.Base64
import org.json.JSONObject
import java.io.File
import java.security.MessageDigest
import java.util.Locale

/**
 * A separate locally pinned Ed25519 catalog signing public key for EACH server
 * that runs its own Remnawave/3x-ui panel and Bridge.
 *
 * NO TRUST ON FIRST USE: the expected fingerprint must be independently
 * confirmed by the operator, not taken from the same HTTP response as the key.
 *
 * Never downloads a key, trusts SNI/URL alone, changes subscription state,
 * enrolls devices, signs anything, or starts VPN/Bridge.
 */
internal class BridgeServerCatalogTrust(
    context: Context,
    // An isolated directory may be supplied only by in-app diagnostics.
    // Production callers always use the app-private noBackupFilesDir default.
    private val baseDir: File = File(
        context.applicationContext.noBackupFilesDir,
        "bridge-server-catalog-trust-v1"
    ),
) {

    data class PinnedKey(
        val serverId: String,
        val keyId: String,
        val publicKeyRaw: ByteArray,
    ) {
        fun fingerprintSha256(): String = sha256Hex(publicKeyRaw)
    }

    private fun requireId(id: String) {
        require(id.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,127}"))) {
            "invalid trusted server ID"
        }
    }

    private fun slot(serverId: String): AtomicFile {
        requireId(serverId)
        return AtomicFile(File(baseDir, sha256Hex(serverId.toByteArray(Charsets.UTF_8)) + ".json"))
    }

    private fun parse(serverId: String, bytes: ByteArray): PinnedKey {
        require(bytes.size in 50..2048) { "invalid pin record length" }
        val json = JSONObject(String(bytes, Charsets.UTF_8))
        require(json.keys().asSequence().toSet() == setOf("format", "server_id", "key_id", "public_key_b64")) {
            "unrecognized pin fields"
        }
        require(json.getString("format") == "tervon.bridge.server-catalog-pin.v1") {
            "unsupported pin format"
        }
        require(json.getString("server_id") == serverId) {
            "pin record belongs to another server"
        }
        val keyId = json.getString("key_id")
        require(keyId.matches(Regex("[A-Za-z0-9_-]{1,64}"))) { "invalid key ID" }
        val encoded = json.getString("public_key_b64")
        require(encoded.length <= 64) { "invalid public key length" }
        val key = Base64.decode(encoded, Base64.NO_WRAP)
        require(key.size == 32 && Base64.encodeToString(key, Base64.NO_WRAP) == encoded) {
            "invalid raw Ed25519 public key"
        }
        return PinnedKey(serverId, keyId, key)
    }

    /**
     * Read an already-confirmed server pin offline; never trust a caller-supplied
     * key as an implicit fallback when no pin exists.
     */
    @Synchronized
    fun pinned(serverId: String): PinnedKey? {
        val stored = slot(serverId)
        if (!stored.baseFile.isFile) return null
        val bytes = stored.openRead().use { stream ->
            val content = stream.readBytes()
            require(content.size <= 2048) { "pin record too large" }
            content
        }
        return parse(serverId, bytes)
    }

    /**
     * Called only from an explicitly confirmed registration interaction.
     *
     * The expected fingerprint MUST be obtained from a trusted independent
     * channel (e.g. authenticated admin interface viewed separately). Supplying
     * the fingerprint from the same unauthenticated server response as rawKey
     * would NOT establish trust.
     *
     * A different key/ID for an existing server is refused, not rotated silently.
     */
    @Synchronized
    fun confirm(
        trustedServerId: String,
        keyId: String,
        publicKeyBase64: String,
        independentlyVerifiedSha256: String,
    ): PinnedKey {
        requireId(trustedServerId)
        require(keyId.matches(Regex("[A-Za-z0-9_-]{1,64}"))) { "invalid signing key ID" }
        require(independentlyVerifiedSha256.matches(Regex("[a-fA-F0-9]{64}"))) {
            "invalid independent public key fingerprint"
        }
        require(publicKeyBase64.length <= 64) { "invalid public key input" }
        val publicKey = Base64.decode(publicKeyBase64, Base64.NO_WRAP)
        require(publicKey.size == 32 &&
            Base64.encodeToString(publicKey, Base64.NO_WRAP) == publicKeyBase64) {
            "invalid Ed25519 public key"
        }
        val calculated = sha256Hex(publicKey)
        require(MessageDigest.isEqual(
            calculated.toByteArray(Charsets.US_ASCII),
            independentlyVerifiedSha256.lowercase(Locale.ROOT).toByteArray(Charsets.US_ASCII),
        )) { "independently verified fingerprint mismatch" }

        val existing = pinned(trustedServerId)
        if (existing != null) {
            require(existing.keyId == keyId && existing.publicKeyRaw.contentEquals(publicKey)) {
                "server signing key changed: explicit separately authorized rotation required"
            }
            return existing
        }

        check(baseDir.isDirectory || baseDir.mkdirs()) { "cannot create private pin directory" }
        val record = JSONObject().apply {
            put("format", "tervon.bridge.server-catalog-pin.v1")
            put("server_id", trustedServerId)
            put("key_id", keyId)
            put("public_key_b64", publicKeyBase64)
        }.toString().toByteArray(Charsets.UTF_8)

        val target = slot(trustedServerId)
        val out = target.startWrite()
        try {
            out.write(record)
            target.finishWrite(out)
        } catch (error: Exception) {
            target.failWrite(out)
            throw error
        }
        return PinnedKey(trustedServerId, keyId, publicKey)
    }

    /** One cached signed catalog per locally trusted server, never a global key. */
    fun openVerifiedCatalog(context: Context, serverId: String): BridgeSignedServerCatalog {
        val pin = pinned(serverId) ?: error("server catalog public key not independently confirmed")
        return BridgeSignedServerCatalog(
            context,
            mapOf(pin.keyId to pin.publicKeyRaw),
            trustedServerId = serverId,
        )
    }

    companion object {
        private fun sha256Hex(bytes: ByteArray): String =
            MessageDigest.getInstance("SHA-256").digest(bytes).joinToString("") {
                "%02x".format(it.toInt() and 0xff)
            }
    }
}
