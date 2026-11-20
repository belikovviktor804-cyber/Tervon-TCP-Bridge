package org.tervon.bridge.smoketest;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import android.widget.TextView;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.File;
import java.util.concurrent.TimeUnit;

public final class SmokeActivity extends Activity {
    private static final String TAG = "TervonBridgeSmoke";
    private TextView output;
    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        output = new TextView(this);
        output.setPadding(32, 48, 32, 32);
        output.setText("Testing packaged Android ELF...\nNo network or VPN changes.");
        setContentView(output);
        new Thread(this::smoke).start();
    }
    private void status(String value) {
        Log.i(TAG, value);
        runOnUiThread(() -> output.setText(value));
    }
    private void smoke() {
        Process process = null;
        try {
            File exe = new File(getApplicationInfo().nativeLibraryDir, "libtervonbridge.so");
            if (!exe.isFile()) {
                status("RESULT=FAIL_BINARY_NOT_EXTRACTED: " + exe);
                return;
            }
            if (!exe.canExecute()) {
                status("RESULT=FAIL_NOT_EXECUTABLE: " + exe);
                return;
            }
            // -h only prints CLI options; it does not start networking, TLS or services.
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
                status("RESULT=PASS_EXEC_ANDROID_ARM64\nELF launched via ProcessBuilder\n-exec path: " + exe + "\nhelp: -config found");
            } else {
                status("RESULT=FAIL_EXIT_OR_HELP\nexit=" + code + "\noutput=" + help.substring(0, Math.min(1200, help.length())));
            }
        } catch (Throwable e) {
            status("RESULT=FAIL_EXCEPTION " + e.getClass().getSimpleName() + ": " + e.getMessage());
        } finally {
            if (process != null && process.isAlive()) process.destroyForcibly();
        }
    }
}
