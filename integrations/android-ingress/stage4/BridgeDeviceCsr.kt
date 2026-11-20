package com.tervonvpndpi.app

import org.bouncycastle.asn1.x500.X500Name
import org.bouncycastle.operator.jcajce.JcaContentSignerBuilder
import org.bouncycastle.pkcs.jcajce.JcaPKCS10CertificationRequestBuilder
import org.bouncycastle.util.io.pem.PemObject
import org.bouncycastle.util.io.pem.PemWriter
import java.io.StringWriter

/**
 * Generates a real PKCS#10 CSR signed by the same P-256 key stored by
 * BridgeDeviceIdentity. Output is standard PEM compatible with the server's
 * Go crypto/x509.ParseCertificateRequest + CheckSignature.
 *
 * Requires a single Android-compatible Bouncy Castle bcpkix dependency,
 * reviewed against the app's current Gradle configuration before merging.
 */
internal class BridgeDeviceCsr(private val identity: BridgeDeviceIdentity) {
    fun createPem(deviceCommonName: String): String {
        require(deviceCommonName.matches(Regex("[A-Za-z0-9][A-Za-z0-9._-]{0,63}"))) {
            "invalid device certificate common name"
        }
        return identity.withKeyPair { publicKey, privateKey ->
            val name = X500Name("CN=$deviceCommonName")
            val builder = JcaPKCS10CertificationRequestBuilder(name, publicKey)
            val signer = JcaContentSignerBuilder("SHA256withECDSA").build(privateKey)
            val request = builder.build(signer)
            StringWriter().use { buffer ->
                PemWriter(buffer).use { writer ->
                    writer.writeObject(PemObject("CERTIFICATE REQUEST", request.encoded))
                }
                buffer.toString()
            }
        }
    }
}
