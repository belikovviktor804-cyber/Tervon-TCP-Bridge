package com.tervonvpndpi.app

/**
 * Portable Kotlin/JVM policy checks: no Android APIs and no network.
 * Execute:
 * kotlinc BridgeServerReconciler.kt BridgeServerReconcilerTest.kt -include-runtime -d /tmp/m6-policy.jar
 * java -jar /tmp/m6-policy.jar
 * (Use a project-private test folder instead of /tmp on Homeserver.)
 */
internal object BridgeServerReconcilerTest {
    @JvmStatic
    fun main(args: Array<String>) {
        val a = BridgeServerReconciler.Server(
            "bridge-A",
            setOf(
                BridgeServerReconciler.Endpoint("a.example", 443),
                BridgeServerReconciler.Endpoint("a.example", 8443)
            ),
            "bridge-a.example"
        )
        val b = BridgeServerReconciler.Server(
            "bridge-B",
            setOf(BridgeServerReconciler.Endpoint("b.example", 443)),
            "bridge-b.example"
        )
        val certA = BridgeServerReconciler.Certificate("bridge-A", 20000)
        val certB = BridgeServerReconciler.Certificate("bridge-B", 20000)
        val known = listOf(a, b)
        val creds = listOf(certA, certB)
        fun checkDecision(
            expected: BridgeServerReconciler.Action,
            on: Boolean,
            endpoint: BridgeServerReconciler.Endpoint?,
            certificates: List<BridgeServerReconciler.Certificate> = creds,
            current: String? = "bridge-A",
            servers: List<BridgeServerReconciler.Server> = known,
            now: Long = 10000,
        ) {
            val actual = BridgeServerReconciler.decide(
                on, endpoint, servers, certificates, current, now
            )
            check(actual.action == expected) {
                "expected $expected, got ${actual.action}"
            }
        }

        val endA = BridgeServerReconciler.Endpoint("A.Example.", 443)
        val endB = BridgeServerReconciler.Endpoint("b.example", 443)
        // Manual OFF: no network work, no enrollment, no profile parsing.
        checkDecision(BridgeServerReconciler.Action.IDLE_OFF, false, null)
        checkDecision(BridgeServerReconciler.Action.IDLE_OFF, false, endB)
        // Same Bridge: do not switch or re-register.
        checkDecision(BridgeServerReconciler.Action.KEEP_CURRENT, true, endA)
        checkDecision(
            BridgeServerReconciler.Action.KEEP_CURRENT, true,
            BridgeServerReconciler.Endpoint("a.example", 8443)
        )
        // Known different Bridge: use existing matching certificate.
        checkDecision(BridgeServerReconciler.Action.SELECT_SAVED, true, endB)
        // No valid certificate: the integration must invoke automatic authorized enrollment.
        checkDecision(
            BridgeServerReconciler.Action.REQUEST_AUTO_ENROLLMENT,
            true, endB, listOf(certA)
        )
        checkDecision(
            BridgeServerReconciler.Action.REQUEST_AUTO_ENROLLMENT,
            true, endB, listOf(certA, certB), now = 20000
        )
        // Unknown/missing/colliding destination fails closed, never falls
        // through to an arbitrary stored certificate.
        checkDecision(
            BridgeServerReconciler.Action.TURN_OFF_UNKNOWN,
            true, BridgeServerReconciler.Endpoint("unrelated.example", 443)
        )
        checkDecision(
            BridgeServerReconciler.Action.TURN_OFF_INVALID_PROFILE, true, null
        )
        checkDecision(
            BridgeServerReconciler.Action.TURN_OFF_INVALID_PROFILE,
            true, BridgeServerReconciler.Endpoint("a.example", 0)
        )
        val collision = b.copy(endpoints = setOf(BridgeServerReconciler.Endpoint("a.example", 443)))
        checkDecision(
            BridgeServerReconciler.Action.TURN_OFF_AMBIGUOUS,
            true, endA, servers = listOf(a, collision)
        )
        // The policy does not resolve DNS or dial; a temporarily unreachable
        // known server is still recognized and retains its certificate.
        checkDecision(BridgeServerReconciler.Action.SELECT_SAVED, true, endB)
        println("M6_OFFLINE_BRIDGE_RECONCILIATION=PASS")
    }
}
