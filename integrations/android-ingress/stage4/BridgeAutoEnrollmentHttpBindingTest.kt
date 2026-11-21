package com.tervonvpndpi.app

/** Focused offline regression for M6 signed-server identity binding.
 * Compile with the actual Stage 4 sources (Android) or with JVM-only
 * stand-ins for Android/JSON and the signed-catalog record.
 * Does not open a network connection or issue a certificate.
 */
internal object BridgeAutoEnrollmentHttpBindingTest {
    @JvmStatic fun main(args: Array<String>) {
        val signed = BridgeSignedServerCatalog.Server(
            id = "stable-server-42",
            endpoints = setOf(BridgeServerReconciler.Endpoint("vpn.example.test", 30000)),
            tlsName = "vp.example.test",
            bridgeUrl = "https://vp.example.test:18443/v2/stream",
            enrollmentUrl = "https://subscriptions.example.test/.well-known/tervon-bridge",
            serverCaPem = "only-a-placeholder-for-an-offline-binding-test",
        )
        val http = BridgeAutoEnrollmentHttp(
            "https://subscriptions.example.test:2096/secret-subscription-path", signed
        )
        check(http.verifiedServerId(signed.bridgeUrl, "vp.example.test") == signed.id)
        check(http.verifiedServerId(signed.bridgeUrl, "VP.EXAMPLE.TEST") == signed.id)
        fun mustReject(block: () -> Unit) {
            var rejected = false
            try { block() } catch (_: IllegalArgumentException) { rejected = true }
            check(rejected) { "untrusted discovery/issuance was accepted" }
        }
        mustReject {
            http.verifiedServerId("https://other.example.test:18443/v2/stream", signed.tlsName)
        }
        mustReject { http.verifiedServerId(signed.bridgeUrl, "other.example.test") }
        mustReject {
            http.issue(
                BridgeAutoCertificateEnrollment.TrustedBridge(signed.tlsName, "client-ca"),
                "X".repeat(100),
            )
        }
        println("M6_SIGNED_SERVER_ID_BINDING=PASS")
    }
}
