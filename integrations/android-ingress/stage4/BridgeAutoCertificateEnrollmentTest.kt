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
        check(coordinator.onBypassEnabled(false, bridge) is BridgeAutoCertificateEnrollment.Outcome.Disabled)
        check(coordinator.onBypassEnabled(true, null) is BridgeAutoCertificateEnrollment.Outcome.NoBridge)
        check(csrCalls == 0 && issueCalls == 0 && installCalls == 0)
        check(coordinator.onBypassEnabled(true, bridge) is BridgeAutoCertificateEnrollment.Outcome.Enrolled)
        check(csrCalls == 1 && issueCalls == 1 && installCalls == 1)
        check(coordinator.onBypassEnabled(true, bridge) is BridgeAutoCertificateEnrollment.Outcome.Reused)
        check(csrCalls == 1 && issueCalls == 1)
        stored = false
        val failed = BridgeAutoCertificateEnrollment(id, storage,
            object : BridgeAutoCertificateEnrollment.EnrollmentTransport {
                override fun issue(trusted: BridgeAutoCertificateEnrollment.TrustedBridge, csrPem: String): BridgeAutoCertificateEnrollment.Issued {
                    error("unreachable")
                }
            })
        check(runCatching { failed.onBypassEnabled(true, bridge) }.isFailure)
        check(!stored)
        println("M6_AUTO_CERT_COORDINATOR=PASS")
    }
}
