package com.tervonvpndpi.app

internal object BridgeAutoCertificateEnrollmentTest {
    @JvmStatic fun main(args: Array<String>) {
        var csrCalls = 0
        var issueCalls = 0
        var installCalls = 0
        var stored = false
        val id = object : BridgeAutoCertificateEnrollment.DeviceIdentity {
            override fun spkiDer() = byteArrayOf(1, 2, 3)
            override fun csrPem(): String { csrCalls++; return "CSR" }
        }
        val storage = object : BridgeAutoCertificateEnrollment.CertificateStorage {
            override fun hasValid(serverId: String, clientCaPem: String, spkiDer: ByteArray) = stored
            override fun install(serverId: String, certificatePem: String, clientCaPem: String, spkiDer: ByteArray) {
                check(serverId == "dynamic.example" && certificatePem == "CERT")
                installCalls++; stored = true
            }
        }
        val transport = object : BridgeAutoCertificateEnrollment.EnrollmentTransport {
            override fun issue(trusted: BridgeAutoCertificateEnrollment.TrustedBridge, csrPem: String): BridgeAutoCertificateEnrollment.Issued {
                issueCalls++; check(csrPem == "CSR")
                return BridgeAutoCertificateEnrollment.Issued("CERT")
            }
        }
        val coordinator = BridgeAutoCertificateEnrollment(id, storage, transport)
        val bridge = BridgeAutoCertificateEnrollment.TrustedBridge("dynamic.example", "trusted CA")
        var discoveries = 0
        val discovery = object : BridgeAutoCertificateEnrollment.SubscriptionBridgeDiscovery {
            override fun discover(activeSubscription: String): BridgeAutoCertificateEnrollment.TrustedBridge? {
                discoveries++
                check(activeSubscription == "https://subscription.example/user/opaque")
                return bridge
            }
        }
        val noBridge = object : BridgeAutoCertificateEnrollment.SubscriptionBridgeDiscovery {
            override fun discover(activeSubscription: String): BridgeAutoCertificateEnrollment.TrustedBridge? {
                discoveries++
                return null
            }
        }
        val subscription = "https://subscription.example/user/opaque"
        check(coordinator.onBypassToggled(false, subscription, discovery) is BridgeAutoCertificateEnrollment.Outcome.Disabled)
        check(coordinator.onBypassToggled(true, null, discovery) is BridgeAutoCertificateEnrollment.Outcome.NoBridge)
        check(discoveries == 0 && csrCalls == 0 && issueCalls == 0 && installCalls == 0)
        check(coordinator.onBypassToggled(true, subscription, noBridge) is BridgeAutoCertificateEnrollment.Outcome.NoBridge)
        check(discoveries == 1 && csrCalls == 0 && issueCalls == 0)
        check(coordinator.onBypassToggled(true, subscription, discovery) is BridgeAutoCertificateEnrollment.Outcome.Enrolled)
        check(discoveries == 2 && csrCalls == 1 && issueCalls == 1 && installCalls == 1)
        check(coordinator.onBypassToggled(true, subscription, discovery) is BridgeAutoCertificateEnrollment.Outcome.Reused)
        check(discoveries == 3 && csrCalls == 1 && issueCalls == 1)
        stored = false
        val failed = BridgeAutoCertificateEnrollment(id, storage,
            object : BridgeAutoCertificateEnrollment.EnrollmentTransport {
                override fun issue(trusted: BridgeAutoCertificateEnrollment.TrustedBridge, csrPem: String): BridgeAutoCertificateEnrollment.Issued {
                    error("unreachable")
                }
            })
        check(runCatching { failed.onBypassToggled(true, subscription, discovery) }.isFailure)
        check(!stored)
        println("M6_AUTO_CERT_COORDINATOR=PASS")
    }
}
