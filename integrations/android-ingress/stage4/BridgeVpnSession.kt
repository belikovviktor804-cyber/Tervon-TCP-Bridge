package com.tervonvpndpi.app

import android.content.Context
import org.json.JSONObject
import java.net.InetSocketAddress
import java.net.InetAddress
import java.net.ServerSocket
import java.net.Socket

/** M6: existing Go v1.0.1 Bridge + existing libv2ray VLESS TCP integration.
 * Never changes the persisted direct Xray config or HEV/TUN. Failures use direct.
 */
internal object BridgeVpnSession {
    private var handoff: BridgeClientPemHandoff? = null
    private var active = false
    private var ownsProcess = false

    @Synchronized
    fun active(): Boolean = active && BridgeRuntimeControl.coreState() == BridgeCoreManager.State.RUNNING

    /** Prepare BEFORE libv2ray startLoop/restart. Call only on the VPN worker. */
    @Synchronized
    fun prepare(context: Context, direct: String): String {
        stop(context)
        val prefs = context.getSharedPreferences("tervon_bridge_settings_v1", Context.MODE_PRIVATE)
        if (!prefs.getBoolean("enabled", false)) return direct
        val subscriptionHost = BridgeSubscriptionBinding.host(context) ?: return direct
        if (!subscriptionHost.matches(Regex("[a-z0-9][a-z0-9.-]{0,127}"))) return direct
        try {
            val root = JSONObject(direct)
            val outbounds = root.getJSONArray("outbounds")
            val proxy = (0 until outbounds.length()).map { outbounds.getJSONObject(it) }
                .single { it.optString("tag") == "proxy" }
            if (proxy.optString("protocol") != "vless") return direct
            val stream = proxy.getJSONObject("streamSettings")
            if (stream.optString("network") != "tcp") return direct
            val dest = proxy.getJSONObject("settings").getJSONArray("vnext").getJSONObject(0)
            val selected = BridgeVlessRoutePlan.Endpoint(dest.getString("address"), dest.getInt("port"))

            // Re-read the approved subscription host on every start/profile change;
            // never use stale discovery parameters from the preceding profile.
            val origin = "https://$subscriptionHost/"
            val http = BridgeAutoEnrollmentHttp(origin)
            val trusted = http.discover(origin) ?: return direct
            val params = http.discoveredParameters() ?: return direct
            if (params.serverId != trusted.serverId) return direct

            val identity = BridgeDeviceIdentity(context)
            val stored = BridgeServerCertificateStore(context).findValid(
                trusted.serverId, params.clientCaPem, identity.publicIdentity().spkiDer
            ) ?: return direct
            // Obtain local listener ports from the OS, not from manually
            // entered values or fixed per-device defaults.
            val port = automaticLoopbackPort()
            val metricsPort = generateSequence { automaticLoopbackPort() }
                .take(5).firstOrNull { it != port } ?: return direct
            handoff = BridgeClientPemHandoff.create(context, identity, stored.certificatePem, params.serverCaPem)
            val files = requireNotNull(handoff)
            val config = BridgeClientConfigBuilder.build(BridgeClientConfigBuilder.Inputs(
                remoteUrl = params.remoteUrl,
                serverName = params.tlsName,
                service = params.service,
                listenPort = port,
                metricsPort = metricsPort,
                clientCertificatePath = files.clientCertFile.absolutePath,
                clientPrivateKeyPath = files.privateKeyFile.absolutePath,
                serverCaPath = files.serverCaFile.absolutePath,
            ))
            val env = BridgeConfigManager(context)
            if (BridgeRuntimeControl.coreState() == BridgeCoreManager.State.RUNNING) {
                stop(context)
                return direct
            }
            env.writeValidatedConfig(config)
            BridgeRuntimeControl.onUnexpectedExit(context) { unexpectedExit(context.applicationContext) }
            check(BridgeRuntimeControl.launchClient(context)) { "Bridge process did not start" }
            ownsProcess = true
            var ready = false
            repeat(25) {
                if (!ready) {
                    try {
                        Socket().use { sock -> sock.connect(InetSocketAddress("127.0.0.1", port), 120) }
                        ready = true
                    } catch (_: Exception) { Thread.sleep(80) }
                }
            }
            check(ready) { "Bridge TCP listener unavailable" }
            // The Go Bridge opens PEM files during startup. Remove handoff after
            // the local listener is confirmed and on every stop/error path.
            files.close()
            handoff = null
            val decision = BridgeVlessRoutePlan.decide(
                enabled = true, protocol = "vless", transport = "tcp",
                activeServer = selected,
                subscriptionVerified = params.serverId == trusted.serverId,
                localPort = port, bridgeReady = ready
            )
            val route = (decision as? BridgeVlessRoutePlan.Decision.ThroughBridge)?.route
                ?: error("VLESS TCP route did not pass authorization")
            val routed = BridgeXrayConfigRoute.apply(direct, route)
            prefs.edit().putInt("metricsPort", metricsPort).apply()
            active = true
            return routed
        } catch (_: Exception) {
            // No certificate, server address or private key in logs/UI.
            stop(context)
            return direct
        }
    }

    private fun automaticLoopbackPort(): Int =
        ServerSocket(0, 0, InetAddress.getLoopbackAddress()).use { it.localPort }

    private fun unexpectedExit(context: Context) {
        val shouldRecover = synchronized(this) {
            if (!active || !ownsProcess) false else {
                active = false
                ownsProcess = false
                true
            }
        }
        if (!shouldRecover) return
        BridgeRuntimeSettings.setEnabled(context, false)
        BridgeSubscriptionBinding.clear(context)
        if (VpnForegroundService.isRunning) {
            VpnForegroundService.restart(context) { ok ->
                if (!ok) VpnForegroundService.stop(context)
            }
        }
    }

    @Synchronized
    fun stop(context: Context) {
        active = false
        if (ownsProcess) {
            BridgeRuntimeControl.onUnexpectedExit(context, null)
            BridgeRuntimeControl.stopClient(context)
            ownsProcess = false
        }
        handoff?.close()
        handoff = null
        BridgeClientPemHandoff.removeAbandoned(context)
    }
}
