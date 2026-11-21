package com.tervonvpndpi.app

import org.json.JSONObject

/**
 * Applies an ALREADY AUTHORIZED, READY, locally listening Bridge route to one
 * generated Xray VLESS TCP outbound. It does not authorize or start Bridge.
 * Never modifies the VLESS UUID, flow, TLS/REALITY parameters, DNS or routing.
 */
internal object BridgeXrayConfigRoute {
    fun apply(configJson: String, route: BridgeVlessRoutePlan.Route): String {
        val root = JSONObject(configJson)
        val outbounds = root.getJSONArray("outbounds")
        val matching = (0 until outbounds.length()).map { outbounds.getJSONObject(it) }
            .filter { it.optString("tag") == "proxy" }
        require(matching.size == 1) { "Exactly one proxy outbound required" }
        val proxy = matching.single()
        require(proxy.optString("protocol") == "vless") { "Only VLESS supported" }
        val stream = proxy.getJSONObject("streamSettings")
        require(stream.optString("network") == "tcp") { "Only VLESS TCP supported" }
        val vnext = proxy.getJSONObject("settings").getJSONArray("vnext")
        require(vnext.length() == 1) { "Exactly one destination required" }
        val destination = vnext.getJSONObject(0)
        require(destination.getString("address").trimEnd('.').equals(
            route.remote.host.trimEnd('.'), ignoreCase = true
        ) && destination.getInt("port") == route.remote.port) {
            "Current VLESS destination does not match the authorized Bridge target"
        }
        require(route.local.host == "127.0.0.1" && route.local.port in 1024..65535) {
            "Bridge route must use verified local IPv4 listener"
        }
        destination.put("address", route.local.host)
        destination.put("port", route.local.port)
        return root.toString()
    }
}
