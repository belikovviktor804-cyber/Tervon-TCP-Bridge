package com.tervonvpndpi.app

import android.content.Context
import org.json.JSONObject
import java.io.BufferedInputStream
import java.net.InetSocketAddress
import java.net.Socket
import java.nio.charset.StandardCharsets
import java.io.File

/** Read-only local monitoring. Does not launch Bridge or change VPN routing. */
object BridgeDiagnostics {
    private const val MAX_RESPONSE_BYTES = 65536
    private const val CONNECT_TIMEOUT_MS = 400
    private const val READ_TIMEOUT_MS = 650
    private const val MAX_METRIC_DEPTH = 4

    fun snapshot(context: Context): Map<String, Any> {
        val bridgeDir = File(context.filesDir, "bridge")
        val configDir = File(bridgeDir, "config")
        val keyDir = File(bridgeDir, "pki")
        val ready = ensurePrivateDir(bridgeDir) &&
            ensurePrivateDir(configDir) && ensurePrivateDir(keyDir)
        val binary = File(context.applicationInfo.nativeLibraryDir, "libtervonbridge.so")
        val base = mutableMapOf<String, Any>(
            "privateEnvironmentReady" to ready,
            "binaryPresent" to (binary.isFile && binary.length() > 0L),
            "binaryBytes" to (if (binary.isFile) binary.length() else 0L),
            // A local metrics response is never proof of remote connectivity.
            "transportConnected" to false,
            "metricsAvailable" to false,
            "metrics" to emptyMap<String, String>(),
            "details" to emptyMap<String, String>(),
            "status" to "Нет данных"
        )
        // Use the EXISTING Bridge settings preference written by BridgeRuntimeSettings.
        // The old bridge_monitoring/metrics_port store was never populated by settings.
        val prefs = context.getSharedPreferences("tervon_bridge_settings_v1", Context.MODE_PRIVATE)
        val port = prefs.getInt("metricsPort", 0)
        if (port !in 1024..65535) {
            base["status"] = "Порт локальной телеметрии не настроен"
            return base
        }
        try {
            val body = readLocalMetrics(port)
            val json = JSONObject(body)
            val metrics = sortedMapOf<String, String>()
            flattenNumeric(json, "", metrics, 0)
            // These are bounded, known local core metadata fields, never secrets.
            val details = sortedMapOf<String, String>()
            for (field in listOf("schema", "build_version", "padding_state", "jitter_state")) {
                val value = json.optString(field, "")
                if (value.length in 1..80 && value.all { it.isLetterOrDigit() || it in "-_." }) {
                    details[field] = value
                }
            }
            base["metricsAvailable"] = true
            base["status"] = "Локальная телеметрия доступна"
            base["metrics"] = metrics
            base["details"] = details
        } catch (_: Exception) {
            base["status"] = "Нет соединения с локальной телеметрией"
        }
        return base
    }

    private fun ensurePrivateDir(dir: File): Boolean {
        if (!dir.isDirectory && !dir.mkdirs()) return false
        return dir.isDirectory && dir.canRead() && dir.canWrite()
    }

    private fun flattenNumeric(obj: JSONObject, prefix: String, output: MutableMap<String, String>, depth: Int) {
        if (depth >= MAX_METRIC_DEPTH || output.size >= 48) return
        val keys = obj.keys()
        while (keys.hasNext() && output.size < 48) {
            val key = keys.next()
            if (!key.matches(Regex("[a-zA-Z0-9_]{1,48}"))) continue
            val value = obj.opt(key)
            val full = if (prefix.isEmpty()) key else "$prefix.$key"
            when (value) {
                is JSONObject -> flattenNumeric(value, full, output, depth + 1)
                is Number -> if (value.toString().length <= 48) output[full] = value.toString()
                is Boolean -> output[full] = value.toString()
            }
        }
    }

    private fun readLocalMetrics(port: Int): String {
        Socket().use { socket ->
            socket.connect(InetSocketAddress("127.0.0.1", port), CONNECT_TIMEOUT_MS)
            socket.soTimeout = READ_TIMEOUT_MS
            socket.getOutputStream().write((
                "GET /metrics/v1 HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\nAccept: application/json\r\n\r\n"
            ).toByteArray(StandardCharsets.US_ASCII))
            socket.getOutputStream().flush()
            val input = BufferedInputStream(socket.getInputStream())
            val bytes = ByteArray(MAX_RESPONSE_BYTES + 1)
            var n = 0
            while (n < bytes.size) {
                val r = input.read(bytes, n, bytes.size - n)
                if (r < 0) break
                n += r
            }
            if (n == bytes.size) throw IllegalStateException("metrics response too large")
            val response = String(bytes, 0, n, StandardCharsets.UTF_8)
            val headEnd = response.indexOf("\r\n\r\n")
            if (headEnd < 0) throw IllegalStateException("malformed HTTP response")
            val status = response.substringBefore("\r\n")
            if (!status.matches(Regex("HTTP/1\\.[01] 200( .*|)"))) throw IllegalStateException("metrics unavailable")
            val headers = response.substring(0, headEnd).lowercase()
            val rawBody = response.substring(headEnd + 4)
            return if (headers.contains("transfer-encoding: chunked")) decodeChunked(rawBody)
                else rawBody
        }
    }

    private fun decodeChunked(body: String): String {
        val out = StringBuilder()
        var index = 0
        while (index < body.length) {
            val lineEnd = body.indexOf("\r\n", index)
            if (lineEnd < 0) throw IllegalStateException("invalid chunk header")
            val length = body.substring(index, lineEnd).substringBefore(';').trim()
                .toIntOrNull(16) ?: throw IllegalStateException("invalid chunk size")
            if (length == 0) return out.toString()
            if (length < 0 || length > MAX_RESPONSE_BYTES || out.length + length > MAX_RESPONSE_BYTES) {
                throw IllegalStateException("chunk limit exceeded")
            }
            index = lineEnd + 2
            if (index + length + 2 > body.length || body.substring(index + length, index + length + 2) != "\r\n") {
                throw IllegalStateException("truncated chunk")
            }
            out.append(body, index, index + length)
            index += length + 2
        }
        throw IllegalStateException("missing terminating chunk")
    }
}
