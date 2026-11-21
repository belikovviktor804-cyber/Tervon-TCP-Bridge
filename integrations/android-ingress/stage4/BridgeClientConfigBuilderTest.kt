package com.tervonvpndpi.app

import org.json.JSONObject

/**
 * Android JVM/instrumented focused test of the v1.0.1 configuration envelope.
 * This source is NOT to be copied into Android app/src/main.
 */
internal object BridgeClientConfigBuilderTest {
    @JvmStatic fun main(args: Array<String>) {
        val valid = BridgeClientConfigBuilder.Inputs(
            remoteUrl = "https://bridge.example.org:443/v2/stream",
            serverName = "bridge.example.org",
            service = "vpn",
            listenPort = 19001,
            clientCertificatePath = "/data/user/0/app/files/bridge/pki/client.crt",
            clientPrivateKeyPath = "/data/user/0/app/files/bridge/pki/client.key",
            serverCaPath = "/data/user/0/app/files/bridge/pki/server-ca.crt",
        )
        val json = JSONObject(BridgeClientConfigBuilder.build(valid))
        check(json.getString("mode") == "client")
        check(json.getString("listen") == "127.0.0.1:19001")
        check(json.getString("remote_url") == valid.remoteUrl)
        check(json.getString("server_name") == valid.serverName)
        check(json.getString("service") == "vpn")
        check(json.getString("cert_file") == valid.clientCertificatePath)
        check(json.getString("key_file") == valid.clientPrivateKeyPath)
        check(json.getString("ca_file") == valid.serverCaPath)
        check(json.getBoolean("padding_enabled"))
        check(json.getBoolean("padding_rate_enabled"))
        check(json.getBoolean("jitter_enabled"))
        check(json.getInt("max_local_connections") == 0)

        fun reject(input: BridgeClientConfigBuilder.Inputs) {
            check(runCatching { BridgeClientConfigBuilder.build(input) }.isFailure)
        }
        reject(valid.copy(remoteUrl = "http://bridge.example.org/v2/stream"))
        reject(valid.copy(remoteUrl = "https://bridge.example.org/v1/stream"))
        reject(valid.copy(remoteUrl = "https://bridge.example.org/v2/stream?x=1"))
        reject(valid.copy(remoteUrl = "https://user@bridge.example.org/v2/stream"))
        reject(valid.copy(listenPort = 0))
        reject(valid.copy(jitterEnabled = true, paddingRateEnabled = false))
        reject(valid.copy(paddingRateEnabled = true, paddingEnabled = false, jitterEnabled = false))
        reject(valid.copy(clientPrivateKeyPath = "relative/key.pem"))
        reject(valid.copy(serverName = "name with spaces"))
        println("M6_BRIDGE_CLIENT_CONFIG_BUILDER=PASS")
    }
}
