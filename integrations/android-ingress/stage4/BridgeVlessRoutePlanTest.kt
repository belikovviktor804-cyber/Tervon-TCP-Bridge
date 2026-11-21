package com.tervonvpndpi.app

internal object BridgeVlessRoutePlanTest {
    @JvmStatic fun main(args: Array<String>) {
        val remote = BridgeVlessRoutePlan.Endpoint("vpn.example", 30000)
        fun plan(
            enabled: Boolean = true,
            protocol: String = "vless",
            transport: String = "tcp",
            active: BridgeVlessRoutePlan.Endpoint? = remote,
            authorized: BridgeVlessRoutePlan.Endpoint? = remote,
            port: Int = 19001,
            ready: Boolean = true
        ) = BridgeVlessRoutePlan.decide(enabled, protocol, transport, active, authorized, port, ready)

        check(plan(enabled = false) is BridgeVlessRoutePlan.Decision.Direct)
        check(plan(protocol = "hysteria2") == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.UNSUPPORTED_TRANSPORT))
        check(plan(transport = "xhttp") == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.UNSUPPORTED_TRANSPORT))
        check(plan(active = null) == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.INVALID_PROFILE))
        check(plan(authorized = null) == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.WRONG_SERVER))
        check(plan(authorized = BridgeVlessRoutePlan.Endpoint("other.example", 30000)) == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.WRONG_SERVER))
        check(plan(port = 0) == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.INVALID_LOCAL_LISTENER))
        check(plan(ready = false) == BridgeVlessRoutePlan.Decision.Blocked(BridgeVlessRoutePlan.Reason.BRIDGE_NOT_READY))
        val routed = plan() as BridgeVlessRoutePlan.Decision.ThroughBridge
        check(routed.route.remote == remote)
        check(routed.route.local == BridgeVlessRoutePlan.Endpoint("127.0.0.1", 19001))
        check(plan(active = BridgeVlessRoutePlan.Endpoint("VPN.EXAMPLE.", 30000)) is BridgeVlessRoutePlan.Decision.ThroughBridge)
        println("M6_VLESS_ROUTE_POLICY=PASS")
    }
}
