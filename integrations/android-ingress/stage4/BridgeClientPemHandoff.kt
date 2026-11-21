package com.tervonvpndpi.app

import android.content.Context
import android.util.Base64
import java.io.File
import java.io.FileOutputStream
import java.nio.charset.StandardCharsets

/**
 * Temporary file handoff for the EXISTING Go Bridge v1.0.1
 * tls.LoadX509KeyPair PEM interface. No additional keys or crypto protocols.
 *
 * Caller must validate client leaf/issuer/key through the existing certificate
 * store, then create these files immediately before Bridge launch. The runtime
 * owner MUST call close() after Bridge startup has loaded TLS material and on
 * every unsuccessful start/stop. Never log PEM or return paths to Flutter.
 *
 * If Android is killed abruptly before cleanup, the next initialization
 * removes only abandoned handoff files within this app-private directory.
 */
internal class BridgeClientPemHandoff private constructor(
    val clientCertFile: File,
    val privateKeyFile: File,
    val serverCaFile: File,
) : AutoCloseable {
    override fun close() {
        privateKeyFile.delete()
        clientCertFile.delete()
        serverCaFile.delete()
    }

    companion object {
        private const val PREFIX = "m6-bridge-handoff-"

        fun create(
            context: Context,
            identity: BridgeDeviceIdentity,
            validatedClientCertPem: String,
            trustedServerCaPem: String,
        ): BridgeClientPemHandoff {
            require(validatedClientCertPem.contains("-----BEGIN CERTIFICATE-----")) {
                "validated client certificate required"
            }
            require(trustedServerCaPem.contains("-----BEGIN CERTIFICATE-----")) {
                "trusted server CA required"
            }
            val env = BridgeConfigManager(context.applicationContext)
            check(env.prepareEnvironment()) { "Bridge private directory unavailable" }
            // Do not touch unrelated files, production certificates or other apps.
            val dir = env.pkiDirectory
            val created = mutableListOf<File>()
            try {
                val cert = createPrivateTemp(dir, ".crt", validatedClientCertPem.toByteArray(StandardCharsets.US_ASCII))
                created.add(cert)
                val serverCa = createPrivateTemp(dir, ".ca", trustedServerCaPem.toByteArray(StandardCharsets.US_ASCII))
                created.add(serverCa)
                val key = identity.withKeyPair { _, privateKey ->
                    val pkcs8 = privateKey.encoded ?: error("device key is not exportable")
                    try {
                        val b64 = Base64.encodeToString(pkcs8, Base64.NO_WRAP)
                        val pem = "-----BEGIN PRIVATE KEY-----\n" +
                            b64.chunked(64).joinToString("\n") +
                            "\n-----END PRIVATE KEY-----\n"
                        createPrivateTemp(dir, ".key", pem.toByteArray(StandardCharsets.US_ASCII))
                    } finally {
                        pkcs8.fill(0)
                    }
                }
                created.add(key)
                return BridgeClientPemHandoff(cert, key, serverCa)
            } catch (e: Exception) {
                created.forEach { it.delete() }
                throw e
            }
        }

        private fun createPrivateTemp(dir: File, suffix: String, data: ByteArray): File {
            val file = File.createTempFile(PREFIX, suffix, dir)
            try {
                require(file.setReadable(false, false) && file.setWritable(false, false)) {
                    "cannot restrict PEM file permissions"
                }
                require(file.setReadable(true, true) && file.setWritable(true, true)) {
                    "cannot set app-only PEM file permissions"
                }
                FileOutputStream(file).use { stream ->
                    stream.write(data)
                    stream.fd.sync()
                }
                return file
            } catch (e: Exception) {
                file.delete()
                throw e
            } finally {
                data.fill(0)
            }
        }

        fun removeAbandoned(context: Context) {
            val dir = BridgeConfigManager(context.applicationContext).pkiDirectory
            if (!dir.isDirectory) return
            // Call only while Bridge is STOPPED, never while running.
            dir.listFiles()?.filter {
                it.isFile && !it.isDirectory && it.name.startsWith(PREFIX) &&
                    (it.name.endsWith(".crt") || it.name.endsWith(".ca") || it.name.endsWith(".key"))
            }?.forEach { it.delete() }
        }
    }
}
