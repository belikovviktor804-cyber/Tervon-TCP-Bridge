package com.tervonvpndpi.app

import android.content.Context
import android.util.Base64
import java.io.File
import java.security.MessageDigest
import java.security.SecureRandom

/**
 * Offline per-server pinning regression check. Uses an isolated temporary
 * directory and RANDOM PUBLIC TEST BYTES ONLY. Never touches production pins,
 * the network, certificates, Xray, the VPN or the key publisher.
 *
 * Call only from explicit diagnostic UI, never at normal app startup.
 */
internal object BridgeServerCatalogTrustSelfCheck {
    fun verify(context: Context): Boolean {
        val root = File(
            context.applicationContext.cacheDir,
            "m6-pin-selfcheck-" + java.util.UUID.randomUUID().toString(),
        )
        check(root.mkdir()) { "cannot create diagnostic storage" }
        try {
            val trust = BridgeServerCatalogTrust(context, root)
            val serverA = "m6-selftest-server-a"
            val serverB = "m6-selftest-server-b"
            check(trust.pinned(serverA) == null && trust.pinned(serverB) == null)

            val publicOne = ByteArray(32).also { SecureRandom().nextBytes(it) }
            val publicTwo = ByteArray(32).also { SecureRandom().nextBytes(it) }
            val encodedOne = Base64.encodeToString(publicOne, Base64.NO_WRAP)
            val encodedTwo = Base64.encodeToString(publicTwo, Base64.NO_WRAP)
            val digestOne = digest(publicOne)
            val digestTwo = digest(publicTwo)

            // A wrong fingerprint must NOT create any pin.
            check(rejected {
                trust.confirm(serverA, "one", encodedOne, digestTwo)
            }) { "mismatching SHA-256 was accepted" }
            check(trust.pinned(serverA) == null) { "failed pin attempt created state" }

            // Independently validated fingerprint pins only the selected server.
            val first = trust.confirm(serverA, "one", encodedOne, digestOne)
            check(first.publicKeyRaw.contentEquals(publicOne))
            check(first.fingerprintSha256() == digestOne)
            check(trust.pinned(serverB) == null) { "pin leaked to another server" }

            // Re-opening the real on-disk record must preserve the same pin.
            val reopened = BridgeServerCatalogTrust(context, root)
            check(reopened.pinned(serverA)?.publicKeyRaw?.contentEquals(publicOne) == true)
            check(reopened.confirm(serverA, "one", encodedOne, digestOne)
                .publicKeyRaw.contentEquals(publicOne)) { "idempotence failed" }

            // Even a separately computed matching digest cannot change the
            // already pinned key or key ID without an explicit rotation process.
            check(rejected {
                reopened.confirm(serverA, "two", encodedTwo, digestTwo)
            }) { "silent replacement accepted" }
            check(rejected {
                reopened.confirm(serverA, "changed_id", encodedOne, digestOne)
            }) { "silent key-id replacement accepted" }
            check(reopened.pinned(serverA)?.publicKeyRaw?.contentEquals(publicOne) == true)

            // Different trusted server IDs must maintain independent pin slots.
            check(reopened.confirm(serverB, "two", encodedTwo, digestTwo)
                .publicKeyRaw.contentEquals(publicTwo))
            check(reopened.pinned(serverA)?.publicKeyRaw?.contentEquals(publicOne) == true)
            return true
        } finally {
            check(root.deleteRecursively()) { "cannot remove test-only pin files" }
        }
    }

    private fun digest(bytes: ByteArray): String =
        MessageDigest.getInstance("SHA-256").digest(bytes).joinToString("") {
            "%02x".format(it.toInt() and 0xff)
        }

    private inline fun rejected(block: () -> Unit): Boolean =
        try {
            block()
            false
        } catch (_: IllegalArgumentException) {
            true
        }
}
