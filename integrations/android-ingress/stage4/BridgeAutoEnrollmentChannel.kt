package com.tervonvpndpi.app

import android.content.Context
import android.os.Handler
import android.os.Looper
import io.flutter.plugin.common.BinaryMessenger
import io.flutter.plugin.common.MethodChannel

/**
 * Triggered only by the user's saved "Обход DPI" ON action.
 * Does not alter Xray, VPN, Bridge services or add extra authentication.
 */
internal object BridgeAutoEnrollmentChannel {
    private const val CHANNEL = "com.tervonvpndpi/bridge_auto_enrollment"
    private val main = Handler(Looper.getMainLooper())

    fun register(messenger: BinaryMessenger, context: Context) {
        val app = context.applicationContext
        MethodChannel(messenger, CHANNEL).setMethodCallHandler { call, result ->
            if (call.method != "onBypassToggled") {
                result.notImplemented()
                return@setMethodCallHandler
            }
            val enabled = call.argument<Boolean>("enabled") == true
            val subscription = call.argument<String>("subscriptionUrl")
            Thread({
                try {
                    val configured = BridgeRuntimeSettings.snapshot(app)["enabled"] == true
                    val outcome = if (!enabled || !configured || subscription.isNullOrBlank()) {
                        if (!enabled || !configured)
                            BridgeAutoCertificateEnrollment.Outcome.Disabled
                        else BridgeAutoCertificateEnrollment.Outcome.NoBridge
                    } else {
                        val network = BridgeAutoEnrollmentHttp(subscription)
                        val identity = BridgeAutoCertificateAndroidAdapter.identity(app, "android-device")
                        val store = BridgeAutoCertificateAndroidAdapter.certificates(app)
                        BridgeAutoCertificateEnrollment(identity, store, network)
                            .onBypassToggled(true, subscription, network)
                    }
                    val status = when (outcome) {
                        is BridgeAutoCertificateEnrollment.Outcome.Disabled -> "DISABLED"
                        is BridgeAutoCertificateEnrollment.Outcome.NoBridge -> "NO_BRIDGE"
                        is BridgeAutoCertificateEnrollment.Outcome.Reused -> "CERTIFICATE_REUSED"
                        is BridgeAutoCertificateEnrollment.Outcome.Enrolled -> "CERTIFICATE_ENROLLED"
                    }
                    main.post { result.success(mapOf("status" to status)) }
                } catch (_: Exception) {
                    // Don't disclose subscription URL, device key, CA or enrollment body.
                    main.post { result.error("BRIDGE_AUTO_ENROLLMENT_FAILED", null, null) }
                }
            }, "bridge-auto-mtls-enrollment").apply { isDaemon = true; start() }
        }
    }
}
