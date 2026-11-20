package com.tervonvpndpi.app

import android.content.Context
import org.bouncycastle.pkcs.PKCS10CertificationRequest
import org.bouncycastle.operator.jcajce.JcaContentVerifierProviderBuilder
import org.bouncycastle.pkcs.jcajce.JcaPKCS10CertificationRequest
import java.security.KeyFactory
import java.security.Signature
import java.security.spec.X509EncodedKeySpec

/**
 * Stage 4 diagnostic, run explicitly during Android instrumentation testing.
 * This does not contact production, write PEM private keys, or install a VPN.
 */
internal object BridgeDeviceSelfCheck {
    fun verify(context: Context): Boolean {
        val first = BridgeDeviceIdentity(context)
        val publicBefore = first.publicIdentity().spkiDer
        val second = BridgeDeviceIdentity(context)
        val publicAfter = second.publicIdentity().spkiDer
        check(publicBefore.contentEquals(publicAfter)) {
            "device identity changed across instances"
        }

        val challenge = ByteArray(32) { it.toByte() }
        val signature = second.signChallenge(challenge)
        val publicKey = KeyFactory.getInstance("EC")
            .generatePublic(X509EncodedKeySpec(publicAfter))
        val verifier = Signature.getInstance("SHA256withECDSA")
        verifier.initVerify(publicKey)
        verifier.update(challenge)
        check(verifier.verify(signature)) { "device challenge proof invalid" }

        val pem = BridgeDeviceCsr(second).createPem("m6-android-self-check")
        val body = pem
            .replace("-----BEGIN CERTIFICATE REQUEST-----", "")
            .replace("-----END CERTIFICATE REQUEST-----", "")
            .filterNot { it.isWhitespace() }
        val der = android.util.Base64.decode(body, android.util.Base64.DEFAULT)
        val request = PKCS10CertificationRequest(der)
        val certifiedPublic = JcaPKCS10CertificationRequest(request).publicKey
        check(certifiedPublic.encoded.contentEquals(publicAfter)) {
            "CSR is not bound to the existing device identity"
        }
        check(request.isSignatureValid(JcaContentVerifierProviderBuilder().build(publicKey))) {
            "CSR signature invalid"
        }
        return true
    }
}
