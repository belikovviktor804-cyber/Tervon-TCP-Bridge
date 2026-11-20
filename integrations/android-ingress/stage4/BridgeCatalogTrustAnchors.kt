package com.tervonvpndpi.app

import android.content.Context
import android.util.Base64
import org.json.JSONObject
import java.io.FileNotFoundException

/**
 * M6: trust roots ONLY from a resource packaged inside the signed Android APK.
 *
 * "bridge_catalog_trust_v1.json" is an APK asset, not a file in app-private
 * storage, a network response or subscription data. It must be provisioned
 * by the APK publisher with genuine Ed25519 PUBLIC keys before release.
 *
 * Missing/malformed asset => hard failure (never TOFU, never an empty trusted
 * catalog, never a fallback to a key advertised by the catalog server).
 */
internal object BridgeCatalogTrustAnchors {
    private const val ASSET = "bridge_catalog_trust_v1.json"
    private const val MAX_BYTES = 4096
    private const val FORMAT = "tervon.bridge.catalog.trust.v1"
    private val keyIdPattern = Regex("[A-Za-z0-9_-]{1,64}")

    fun openVerifiedCatalog(context: Context): BridgeSignedServerCatalog {
        return BridgeSignedServerCatalog(context, loadPinnedKeys(context))
    }

    fun loadPinnedKeys(context: Context): Map<String, ByteArray> {
        val bytes = try {
            context.applicationContext.assets.open(ASSET).use { stream ->
                // Limit input even if an accidentally packaged asset is huge.
                val data = ByteArray(MAX_BYTES + 1)
                var total = 0
                while (total < data.size) {
                    val count = stream.read(data, total, data.size - total)
                    if (count < 0) break
                    if (count == 0) continue
                    total += count
                }
                require(total in 1..MAX_BYTES) {
                    "invalid packaged Bridge catalog trust asset size"
                }
                data.copyOf(total)
            }
        } catch (_: FileNotFoundException) {
            error("trusted Bridge catalog public keys are not bundled in this APK")
        }
        val json = JSONObject(String(bytes, Charsets.UTF_8))
        require(json.keys().asSequence().toSet() == setOf("format", "keys")) {
            "invalid Bridge trust asset schema"
        }
        require(json.getString("format") == FORMAT) {
            "unsupported Bridge trust asset format"
        }

        val keys = json.getJSONObject("keys")
        val ids = keys.keys().asSequence().toList()
        require(ids.size in 1..4) { "invalid pinned Bridge key count" }
        val output = LinkedHashMap<String, ByteArray>()
        for (keyId in ids) {
            require(keyIdPattern.matches(keyId)) { "invalid pinned key ID" }
            val encoded = keys.getString(keyId)
            require(encoded.length <= 64) { "invalid pinned public key encoding" }
            val decoded = Base64.decode(encoded, Base64.NO_WRAP)
            require(decoded.size == 32 &&
                Base64.encodeToString(decoded, Base64.NO_WRAP) == encoded) {
                "invalid pinned Ed25519 public key"
            }
            require(output.values.none { it.contentEquals(decoded) }) {
                "duplicate pinned public key"
            }
            output[keyId] = decoded
        }
        return output
    }
}
