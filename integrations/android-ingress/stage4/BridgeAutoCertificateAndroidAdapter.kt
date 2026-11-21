package com.tervonvpndpi.app

import android.content.Context

/**
 * Adapter of the existing Stage 4 identity/CSR and certificate storage.
 * Deliberately contains no enrollment URL, hardcoded domain or new crypto.
 */
internal object BridgeAutoCertificateAndroidAdapter {
    fun identity(context: Context, deviceCommonName: String): BridgeAutoCertificateEnrollment.DeviceIdentity {
        require(deviceCommonName.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,63}")))
        val existing = BridgeDeviceIdentity(context.applicationContext)
        val existingCsr = BridgeDeviceCsr(existing)
        return object : BridgeAutoCertificateEnrollment.DeviceIdentity {
            override fun spkiDer(): ByteArray = existing.publicIdentity().spkiDer
            override fun csrPem(): String = existingCsr.createPem(deviceCommonName)
        }
    }

    fun certificates(context: Context): BridgeAutoCertificateEnrollment.CertificateStorage {
        val existing = BridgeServerCertificateStore(context.applicationContext)
        return object : BridgeAutoCertificateEnrollment.CertificateStorage {
            override fun hasValid(serverId: String, clientCaPem: String, spkiDer: ByteArray): Boolean {
                return existing.findValid(serverId, clientCaPem, spkiDer) != null
            }

            override fun install(serverId: String, certificatePem: String, clientCaPem: String, spkiDer: ByteArray) {
                existing.install(serverId, certificatePem, clientCaPem, spkiDer)
            }
        }
    }
}
