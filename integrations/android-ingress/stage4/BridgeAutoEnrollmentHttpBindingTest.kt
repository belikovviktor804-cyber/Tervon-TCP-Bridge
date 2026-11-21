package com.tervonvpndpi.app

/** Offline policy regression: the selected HTTPS subscription owns discovery.
 * No manual Ed25519 pin, signed catalog or automatic trust-on-first-use.
 * This test does not issue certificates or connect to a network.
 */
internal object BridgeAutoEnrollmentHttpBindingTest {
    @JvmStatic fun main(args: Array<String>) {
        val client = BridgeAutoEnrollmentHttp(
            "https://Subscriptions.Example.test:2096/secret-subscription-path?token=private"
        )
        check(client.discoveredParameters() == null)
        check(client.verifiedServerId(
            "https://bridge.example.test:18443/v2/stream", "bridge.example.test"
        ) == "subscriptions.example.test")
        check(client.verifiedServerId(
            "https://bridge.example.test:18443/v2/stream", "BRIDGE.EXAMPLE.TEST"
        ) == "subscriptions.example.test")
        fun mustReject(block: () -> Unit) {
            var rejected = false
            try { block() } catch (_: IllegalArgumentException) { rejected = true }
            check(rejected) { "insecure enrollment binding accepted" }
        }
        mustReject {
            client.verifiedServerId("http://bridge.example.test/v2/stream", "bridge.example.test")
        }
        mustReject {
            client.verifiedServerId("https://bridge.example.test/v2/stream", "different.example.test")
        }
        mustReject {
            client.verifiedServerId("https://bridge.example.test/v2/stream?test=1", "bridge.example.test")
        }
        mustReject {
            BridgeAutoEnrollmentHttp("http://subscriptions.example.test:2096/secret")
        }
        // POST cannot precede a successful HTTPS discovery, even with a valid CSR shape.
        var rejected = false
        try {
            client.issue(
                BridgeAutoCertificateEnrollment.TrustedBridge(
                    "subscriptions.example.test", "not-a-client-CA"
                ),
                "X".repeat(100),
            )
        } catch (_: IllegalStateException) { rejected = true }
        check(rejected) { "certificate issued without verified HTTPS discovery" }
        println("M6_HTTPS_SUBSCRIPTION_BINDING=PASS")
    }
}
