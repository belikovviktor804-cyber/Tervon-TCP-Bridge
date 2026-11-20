package org.tervon.bridge.smoketest;

import android.app.Activity;
import android.os.Bundle;
import android.graphics.Color;
import android.graphics.Typeface;
import android.util.Log;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.WindowInsets;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Button;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.File;
import java.util.concurrent.TimeUnit;

public final class SmokeActivity extends Activity {
    private static final String TAG = "TervonBridgeSmoke";
    private TextView output;
    private TextView header;

    private int dp(float v) {
        return (int)(v * getResources().getDisplayMetrics().density + 0.5f);
    }

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        getWindow().setStatusBarColor(Color.rgb(11, 18, 29));
        getWindow().setNavigationBarColor(Color.rgb(11, 18, 29));
        getWindow().getDecorView().setSystemUiVisibility(0);

        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setBackgroundColor(Color.rgb(248, 250, 252));
        root.setPadding(dp(20), dp(24), dp(20), dp(24));

        header = new TextView(this);
        header.setText("Tervon Bridge — Android ELF launch test");
        header.setTextSize(TypedValue.COMPLEX_UNIT_SP, 22);
        header.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        header.setTextColor(Color.rgb(11, 18, 29));
        header.setGravity(Gravity.START);
        root.addView(header, new LinearLayout.LayoutParams(-1, -2));

        TextView description = new TextView(this);
        description.setText("Test 1: executable launch. Test 2: local-only HTTPS/2 + mTLS TCP. No VPN or remote servers.\n");
        description.setTextSize(TypedValue.COMPLEX_UNIT_SP, 14);
        description.setTextColor(Color.rgb(75, 85, 99));
        description.setPadding(0, dp(12), 0, dp(12));
        root.addView(description, new LinearLayout.LayoutParams(-1, -2));

        Button loopbackButton = new Button(this);
        loopbackButton.setText("Run local TCP / TLS 1.3 / mTLS test");
        loopbackButton.setAllCaps(false);
        root.addView(loopbackButton, new LinearLayout.LayoutParams(-1, dp(56)));
        loopbackButton.setOnClickListener(v -> {
            loopbackButton.setEnabled(false);
            status("RESULT=RUNNING_LOOPBACK\\nTwo Bridge processes on 127.0.0.1...");
            new Thread(() -> {
                String result = LoopbackTest.run(getApplicationContext());
                status(result);
                runOnUiThread(() -> loopbackButton.setEnabled(true));
            }, "m6-loopback-test").start();
        });

        ScrollView scroll = new ScrollView(this);
        scroll.setFillViewport(true);
        output = new TextView(this);
        output.setTextColor(Color.rgb(18, 45, 38));
        output.setBackgroundColor(Color.rgb(227, 246, 235));
        output.setTextSize(TypedValue.COMPLEX_UNIT_SP, 18);
        output.setTypeface(Typeface.MONOSPACE);
        output.setTextIsSelectable(true);
        output.setPadding(dp(14), dp(20), dp(14), dp(20));
        output.setText("RESULT=RUNNING\nChecking nativeLibraryDir...\n");
        scroll.addView(output, new ScrollView.LayoutParams(-1, -2));
        root.addView(scroll, new LinearLayout.LayoutParams(-1, 0, 1f));
        // Android 15+ can draw behind system bars; preserve top/bottom safe areas.
        root.setOnApplyWindowInsetsListener((v, insets) -> {
            int bars = WindowInsets.Type.statusBars() | WindowInsets.Type.navigationBars();
            android.graphics.Insets safe = insets.getInsets(bars);
            v.setPadding(dp(20), dp(24) + safe.top, dp(20), dp(24) + safe.bottom);
            return insets;
        });
        setContentView(root);
        new Thread(this::smoke, "bridge-elf-smoke").start();
    }

    private void status(String value) {
        Log.i(TAG, value);
        runOnUiThread(() -> {
            output.setText(value);
            output.setBackgroundColor(value.startsWith("RESULT=PASS")
                ? Color.rgb(221, 246, 229) : Color.rgb(255, 238, 227));
        });
    }

    private void smoke() {
        Process process = null;
        try {
            File exe = new File(getApplicationInfo().nativeLibraryDir, "libtervonbridge.so");
            if (!exe.isFile()) {
                status("RESULT=FAIL_BINARY_NOT_EXTRACTED\n" + exe);
                return;
            }
            if (!exe.canExecute()) {
                status("RESULT=FAIL_NOT_EXECUTABLE\n" + exe);
                return;
            }
            // -h only prints CLI usage; it never starts network or VPN operations.
            process = new ProcessBuilder(exe.getAbsolutePath(), "-h")
                .redirectErrorStream(true)
                .start();
            if (!process.waitFor(10, TimeUnit.SECONDS)) {
                process.destroyForcibly();
                status("RESULT=FAIL_TIMEOUT");
                return;
            }
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            try (InputStream in = process.getInputStream()) {
                byte[] buf = new byte[4096];
                int read;
                while ((read = in.read(buf)) != -1 && out.size() < 64000) {
                    out.write(buf, 0, read);
                }
            }
            String help = out.toString("UTF-8");
            int code = process.exitValue();
            if (code == 0 && help.contains("-config")) {
                status("RESULT=PASS_EXEC_ANDROID_ARM64\nELF launched via ProcessBuilder\nPath: " + exe + "\nhelp: -config found");
            } else {
                status("RESULT=FAIL_EXIT_OR_HELP\nexit=" + code + "\noutput=" + help.substring(0, Math.min(1200, help.length())));
            }
        } catch (Throwable e) {
            status("RESULT=FAIL_EXCEPTION\n" + e.getClass().getSimpleName() + ": " + e.getMessage());
        } finally {
            if (process != null && process.isAlive()) process.destroyForcibly();
        }
    }
}
