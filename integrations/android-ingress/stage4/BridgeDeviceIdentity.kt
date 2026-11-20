package com.tervonvpndpi.app

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.AtomicFile
import android.util.Base64
import org.json.JSONObject
import java.io.File
import java.security.KeyFactory
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.PrivateKey
import java.security.PublicKey
import java.security.SecureRandom
import java.security.Signature
import java.security.interfaces.ECPrivateKey
import java.security.interfaces.ECPublicKey
import java.security.spec.ECGenParameterSpec
import java.security.spec.PKCS8EncodedKeySpec
import java.security.spec.X509EncodedKeySpec
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * Stage 4 source candidate. No side effects until invoked by the Android app.
 *
 * Go Bridge v1.0.1 loads PEM keys from app-private files. Its separate ELF process
 * cannot invoke Android Keystore's crypto.Signer without new IPC/JNI. Therefore this
 * class encrypts a per-install exportable P-256 PKCS#8 key at rest using a
 * non-exportable Android Keystore AES-GCM wrapping key.
 *
 * A private-key PEM may only be materialized for the runtime immediately before
 * starting Bridge and must be removed when that process exits. This class itself
 * does not write PEM or invoke the Bridge process.
 */
internal class BridgeDeviceIdentity(context: Context) {
    private val store = context.applicationContext
    private val file = AtomicFile(File(store.noBackupFilesDir, "tervon-bridge-device-identity.json"))
    private val aesAlias = "${store.packageName}.tervon.bridge.device.wrap.v1"

    internal data class PublicIdentity(val spkiDer: ByteArray) {
        fun spkiBase64(): String = Base64.encodeToString(spkiDer, Base64.NO_WRAP)
    }

    @Synchronized
    fun publicIdentity(): PublicIdentity {
        val current = loadOrCreate()
        return PublicIdentity(current.publicKey.encoded.copyOf())
    }

    /** Challenge must come from a fresh, authenticated enrollment session. */
    @Synchronized
    fun signChallenge(challenge: ByteArray): ByteArray {
        require(challenge.size in 32..4096) { "invalid enrollment challenge length" }
        val current = loadOrCreate()
        val signer = Signature.getInstance("SHA256withECDSA")
        signer.initSign(current.privateKey)
        signer.update(challenge)
        return signer.sign()
    }

    /**
     * Restrict private key access to a synchronous, in-process operation.
     * This is used by BridgeDeviceCsr and the future short-lived PEM handoff.
     */
    @Synchronized
    internal fun <T> withKeyPair(block: (PublicKey, PrivateKey) -> T): T {
        val current = loadOrCreate()
        return block(current.publicKey, current.privateKey)
    }

    private data class KeyPairData(val publicKey: PublicKey, val privateKey: PrivateKey)

    private fun loadOrCreate(): KeyPairData {
        val identityFile = file.baseFile
        if (!identityFile.exists()) {
            val generator = KeyPairGenerator.getInstance("EC")
            generator.initialize(ECGenParameterSpec("secp256r1"), SecureRandom())
            val pair = generator.generateKeyPair()
            val secret = getOrCreateWrappingKey()
            val encrypted = encrypt(secret, pair.private.encoded)
            val record = JSONObject()
                .put("version", 1)
                .put("publicSpki", encode(pair.public.encoded))
                .put("iv", encode(encrypted.first))
                .put("ciphertext", encode(encrypted.second))
            val stream = file.startWrite()
            try {
                stream.write(record.toString().toByteArray(Charsets.UTF_8))
                file.finishWrite(stream)
            } catch (e: Exception) {
                file.failWrite(stream)
                throw e
            }
            return KeyPairData(pair.public, pair.private)
        }

        val raw = file.openRead().use {
            val bytes = it.readBytes()
            require(bytes.size in 1..16384) { "invalid device identity file size" }
            String(bytes, Charsets.UTF_8)
        }
        val record = JSONObject(raw)
        require(record.getInt("version") == 1) { "unsupported identity storage version" }
        val iv = decode(record.getString("iv"))
        val ciphertext = decode(record.getString("ciphertext"))
        val spki = decode(record.getString("publicSpki"))
        require(iv.size == 12 && ciphertext.size in 17..8192 && spki.size in 32..4096) {
            "invalid stored device identity"
        }

        // Never silently create a new AES key when a prior encrypted identity exists.
        val secret = getWrappingKey() ?: error("device identity wrapping key unavailable")
        val privateDer = decrypt(secret, iv, ciphertext)
        try {
            val privateKey = KeyFactory.getInstance("EC").generatePrivate(PKCS8EncodedKeySpec(privateDer))
            val publicKey = KeyFactory.getInstance("EC").generatePublic(X509EncodedKeySpec(spki))
            require(privateKey is ECPrivateKey && publicKey is ECPublicKey) { "invalid device EC keys" }
            require(privateKey.params.curve.field.fieldSize == 256 &&
                    publicKey.params.curve.field.fieldSize == 256) { "device keys are not P-256" }
            // Detect mismatch between separately stored SPKI and private PKCS#8.
            val proof = ByteArray(32).also { SecureRandom().nextBytes(it) }
            val signature = Signature.getInstance("SHA256withECDSA").apply {
                initSign(privateKey)
                update(proof)
            }.sign()
            val verifier = Signature.getInstance("SHA256withECDSA").apply {
                initVerify(publicKey)
                update(proof)
            }
            require(verifier.verify(signature)) { "device identity key pair mismatch" }
            return KeyPairData(publicKey, privateKey)
        } finally {
            privateDer.fill(0)
        }
    }

    private fun getOrCreateWrappingKey(): SecretKey {
        getWrappingKey()?.let { return it }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore")
        generator.init(
            KeyGenParameterSpec.Builder(
                aesAlias, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setRandomizedEncryptionRequired(true)
                .build()
        )
        return generator.generateKey()
    }

    private fun getWrappingKey(): SecretKey? {
        val keystore = KeyStore.getInstance("AndroidKeyStore")
        keystore.load(null)
        return keystore.getKey(aesAlias, null) as? SecretKey
    }

    private fun encrypt(secret: SecretKey, data: ByteArray): Pair<ByteArray, ByteArray> {
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.ENCRYPT_MODE, secret)
        cipher.updateAAD("tervon-bridge-identity-v1".toByteArray(Charsets.UTF_8))
        return cipher.iv to cipher.doFinal(data)
    }

    private fun decrypt(secret: SecretKey, iv: ByteArray, data: ByteArray): ByteArray {
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.DECRYPT_MODE, secret, GCMParameterSpec(128, iv))
        cipher.updateAAD("tervon-bridge-identity-v1".toByteArray(Charsets.UTF_8))
        return cipher.doFinal(data)
    }

    private fun encode(data: ByteArray): String = Base64.encodeToString(data, Base64.NO_WRAP)
    private fun decode(text: String): ByteArray = Base64.decode(text, Base64.NO_WRAP)
}
