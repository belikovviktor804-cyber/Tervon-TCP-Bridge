package com.tervonvpndpi.app

import org.json.JSONObject
import java.net.URI

/**
 * Configures the existing Bridge v1.0.1 client; no new transport or cryptography.
 * Call only with independently authorized server metadata, a validated mTLS
 * certificate matching the Stage 4 identity, and app-private PEM file paths.
 *
 * This class does not issue a certificate, write private keys, start Bridge,
 * or switch a VPN route.
 */
internal object BridgeClientConfigBuilder {
    data class Inputs(
        val remoteUrl: String,
        val serverName: String,
        val service: String,
        val listenPort: Int,
        val clientCertificatePath: String,
        val clientPrivateKeyPath: String,
        val serverCaPath: String,
        val paddingEnabled: Boolean = true,
        val paddingRateEnabled: Boolean = true,
        val jitterEnabled: Boolean = true,
    )

    fun build(input: Inputs): String {
        require(input.listenPort in 1024..65535) { "invalid local listener" }
        require(input.serverName.isNotBlank() &&
                input.serverName.length <= 253 &&
                input.serverName.all { it.isLetterOrDigit() || it in ".-" }) {
            "invalid Bridge TLS name"
        }
        require(input.service.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,63}"))) {
            "invalid Bridge service"
        }
        val url = URI(input.remoteUrl)
        require(url.scheme == "https" && url.host != null &&
                url.host.isNotBlank() && url.rawUserInfo == null &&
                url.rawQuery == null && url.rawFragment == null &&
                url.rawPath == "/v2/stream" && url.rawAuthority != null) {
            "invalid Bridge v1.0.1 HTTPS endpoint"
        }
        require(!input.jitterEnabled || input.paddingRateEnabled) {
            "jitter requires rate padding"
        }
        require(!input.paddingRateEnabled || input.paddingEnabled) {
            "rate padding requires padding"
        }
        val paths = listOf(
            input.clientCertificatePath,
            input.clientPrivateKeyPath,
            input.serverCaPath,
        )
        require(paths.all { it.startsWith("/") && it.length <= 1024 &&
                '\u0000' !in it && '\n' !in it && '\r' !in it }) {
            "invalid local PEM path"
        }

        return JSONObject().apply {
            put("mode", "client")
            put("listen", "127.0.0.1:${input.listenPort}")
            put("remote_url", input.remoteUrl)
            put("service", input.service)
            put("server_name", input.serverName)
            put("cert_file", input.clientCertificatePath)
            put("key_file", input.clientPrivateKeyPath)
            put("ca_file", input.serverCaPath)
            put("max_local_connections", 0)
            put("padding_enabled", input.paddingEnabled)
            put("padding_rate_enabled", input.paddingRateEnabled)
            put("jitter_enabled", input.jitterEnabled)
        }.toString()
    }
}
