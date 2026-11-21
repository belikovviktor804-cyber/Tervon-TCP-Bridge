package com.tervonvpndpi.app

import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.net.HttpURLConnection
import java.net.URI
import java.net.URL
import javax.net.ssl.HttpsURLConnection

/**
 * Existing-subscription-domain discovery; no hardcoded server address,
 * client account, user-registration secret, or new crypto.
 *
 * HTTPS is authenticated by Android's platform trust store and hostname
 * verification. The discovery server determines whether Bridge is configured
 * and provides its own existing CA and client parameters.
 */
internal class BridgeAutoEnrollmentHttp(private val subscriptionUrl: String) :
    BridgeAutoCertificateEnrollment.SubscriptionBridgeDiscovery,
    BridgeAutoCertificateEnrollment.EnrollmentTransport {

    private val endpoint: URL = run {
        val origin = URI(subscriptionUrl.trim())
        require(origin.scheme.equals("https", ignoreCase = true) &&
                !origin.host.isNullOrBlank() && origin.rawUserInfo == null &&
                origin.port in -1..65535 && origin.port != 0 &&
                origin.rawFragment == null) {
            "current subscription must be an HTTPS URL"
        }
        // Fixed protocol path, NOT a fixed server; never forward subscription
        // token path or query parameters to Bridge discovery/enrollment.
        URL("https", origin.host, if (origin.port == -1) 443 else origin.port,
            "/.well-known/tervon-bridge")
    }

    override fun discover(activeSubscription: String): BridgeAutoCertificateEnrollment.TrustedBridge? {
        require(activeSubscription == subscriptionUrl) { "subscription changed during registration" }
        val (code, json) = request("GET", null)
        if (code == HttpURLConnection.HTTP_NOT_FOUND) return null
        require(code == HttpURLConnection.HTTP_OK) { "Bridge discovery unavailable" }
        val data = JSONObject(json)
        if (!data.optBoolean("available", false)) return null
        val remote = URI(data.getString("remote_url"))
        require(remote.scheme == "https" && remote.host != null &&
                remote.rawUserInfo == null && remote.rawQuery == null &&
                remote.rawFragment == null && remote.rawPath == "/v2/stream") {
            "invalid server Bridge parameters"
        }
        require(data.getString("service") == "vpn") { "unsupported Bridge service" }
        val name = data.getString("server_name")
        require(name.equals(remote.host, ignoreCase = true)) {
            "Bridge TLS identity differs from advertised endpoint"
        }
        return BridgeAutoCertificateEnrollment.TrustedBridge(
            name, data.getString("client_ca")
        )
    }

    override fun issue(
        trusted: BridgeAutoCertificateEnrollment.TrustedBridge,
        csrPem: String
    ): BridgeAutoCertificateEnrollment.Issued {
        require(csrPem.length in 64..8192) { "invalid CSR" }
        val (code, response) = request("POST", csrPem.toByteArray(Charsets.US_ASCII))
        require(code == HttpURLConnection.HTTP_OK) { "Bridge enrollment unavailable" }
        val certificate = JSONObject(response).getString("certificate")
        require(certificate.length in 100..32768) { "invalid issued certificate size" }
        // Existing BridgeServerCertificateStore validates signer, clientAuth,
        // expiry and exact device public key before writing this certificate.
        return BridgeAutoCertificateEnrollment.Issued(certificate)
    }

    private fun request(method: String, body: ByteArray?): Pair<Int, String> {
        val connection = endpoint.openConnection() as HttpsURLConnection
        try {
            connection.requestMethod = method
            connection.instanceFollowRedirects = false
            connection.connectTimeout = 7000
            connection.readTimeout = 7000
            connection.setRequestProperty("Accept", "application/json")
            if (body != null) {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/pkcs10")
                connection.outputStream.use { it.write(body) }
            }
            val code = connection.responseCode
            if (code == HttpURLConnection.HTTP_NOT_FOUND) return code to ""
            // Do not read error bodies (may contain server paths or secrets).
            if (code != HttpURLConnection.HTTP_OK) return code to ""
            val text = connection.inputStream.use { input ->
                val output = ByteArrayOutputStream()
                val buf = ByteArray(4096)
                while (true) {
                    val read = input.read(buf)
                    if (read < 0) break
                    require(output.size() + read <= 65536) { "Bridge response too large" }
                    output.write(buf, 0, read)
                }
                output.toString("UTF-8")
            }
            return code to text
        } finally {
            connection.disconnect()
        }
    }
}
