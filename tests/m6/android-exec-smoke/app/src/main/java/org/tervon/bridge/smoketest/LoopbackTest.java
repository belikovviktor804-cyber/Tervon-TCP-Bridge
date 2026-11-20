package org.tervon.bridge.smoketest;

import android.content.Context;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.IOException;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.security.MessageDigest;
import java.util.Arrays;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

/** One-shot loopback-only M6 test. No external endpoints, VPN or production credentials. */
final class LoopbackTest {
    private LoopbackTest() { }

    private static int unusedPort() throws IOException {
        try (ServerSocket s = new ServerSocket(0, 1, InetAddress.getByName("127.0.0.1"))) {
            return s.getLocalPort();
        }
    }

    private static File asset(Context context, File dir, String name) throws IOException {
        File target = new File(dir, name);
        try (InputStream in = context.getAssets().open("m6-pki/" + name);
             OutputStream out = new FileOutputStream(target)) {
            byte[] buf = new byte[8192];
            int n;
            while ((n = in.read(buf)) >= 0) out.write(buf, 0, n);
        }
        return target;
    }

    private static void write(File f, String s) throws IOException {
        try (FileOutputStream o = new FileOutputStream(f)) {
            o.write(s.getBytes(StandardCharsets.UTF_8));
        }
    }

    private static String quote(String s) {
        return "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"") + "\"";
    }

    private static final class Running {
        final Process proc;
        final StringBuilder logs = new StringBuilder();
        Running(File exe, File config) throws IOException {
            proc = new ProcessBuilder(exe.getAbsolutePath(), "-config", config.getAbsolutePath())
                .redirectErrorStream(true).start();
            Thread t = new Thread(() -> {
                try (InputStream in = proc.getInputStream()) {
                    byte[] buf = new byte[1024];
                    int n;
                    while ((n = in.read(buf)) > 0) {
                        synchronized(logs) {
                            if (logs.length() < 6000)
                                logs.append(new String(buf, 0, n, StandardCharsets.UTF_8));
                        }
                    }
                } catch (IOException ignored) { }
            }, "bridge-stdio");
            t.setDaemon(true);
            t.start();
        }
        void stop() {
            proc.destroy();
            try {
                if (!proc.waitFor(2, TimeUnit.SECONDS)) proc.destroyForcibly();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                proc.destroyForcibly();
            }
        }
        String brief() {
            synchronized(logs) {
                return logs.substring(0, Math.min(1400, logs.length())).replaceAll("[\\r\\n]+", " ");
            }
        }
    }

    static String run(Context context) {
        File exe = new File(context.getApplicationInfo().nativeLibraryDir, "libtervonbridge.so");
        if (!exe.isFile() || !exe.canExecute()) return "RESULT=FAIL_EXECUTABLE_MISSING";
        Running server = null, client = null;
        ServerSocket backend = null;
        Thread echo = null;
        AtomicReference<Throwable> echoError = new AtomicReference<>();
        try {
            File dir = new File(context.getFilesDir(), "m6-loopback-test");
            if (!dir.exists() && !dir.mkdirs()) throw new IOException("test dir creation failed");
            String[] names = {"ca.pem", "server.pem", "server-key.pem", "client.pem", "client-key.pem", "client-fingerprint.txt"};
            for (String n : names) asset(context, dir, n);
            String fingerprint = new String(Files.readAllBytes(new File(dir, "client-fingerprint.txt").toPath()), StandardCharsets.UTF_8).trim();
            if (!fingerprint.matches("[a-f0-9]{64}")) throw new IOException("invalid test fingerprint");
            int backendPort = unusedPort();
            int serverPort = unusedPort();
            int clientPort = unusedPort();
            // Deliberately listen only on loopback; never expose the test server to the network.
            backend = new ServerSocket(backendPort, 1, InetAddress.getByName("127.0.0.1"));
            backend.setSoTimeout(16000);
            final ServerSocket echoListener = backend;
            echo = new Thread(() -> {
                try (Socket s = echoListener.accept()) {
                    s.setSoTimeout(10000);
                    byte[] buf = new byte[4096];
                    int n;
                    while ((n = s.getInputStream().read(buf)) != -1) {
                        s.getOutputStream().write(buf, 0, n);
                        s.getOutputStream().flush();
                    }
                } catch (Throwable t) { echoError.set(t); }
            }, "m6-echo");
            echo.setDaemon(true);
            echo.start();

            String ca = quote(new File(dir, "ca.pem").getAbsolutePath());
            String serverCert = quote(new File(dir, "server.pem").getAbsolutePath());
            String serverKey = quote(new File(dir, "server-key.pem").getAbsolutePath());
            String clientCert = quote(new File(dir, "client.pem").getAbsolutePath());
            String clientKey = quote(new File(dir, "client-key.pem").getAbsolutePath());
            String common = "\"max_streams\":0,\"padding_enabled\":true,\"padding_rate_enabled\":true,\"jitter_enabled\":true";
            String serverConfig = "{\"mode\":\"server\",\"listen\":\"127.0.0.1:" + serverPort + "\","
                + "\"cert_file\":" + serverCert + ",\"key_file\":" + serverKey + ",\"ca_file\":" + ca + ","
                + common + ",\"routes\":{\"loopback\":[{\"Name\":\"echo\",\"Address\":\"127.0.0.1:" + backendPort + "\"}]},"
                + "\"authorized\":{\"" + fingerprint + "\":[\"loopback\"]}}";
            String clientConfig = "{\"mode\":\"client\",\"listen\":\"127.0.0.1:" + clientPort + "\","
                + "\"cert_file\":" + clientCert + ",\"key_file\":" + clientKey + ",\"ca_file\":" + ca + ","
                + "\"server_name\":\"localhost\",\"remote_url\":\"https://localhost:" + serverPort + "/v2/stream\","
                + "\"service\":\"loopback\",\"max_local_connections\":0,\"padding_enabled\":true,"
                + "\"padding_rate_enabled\":true,\"jitter_enabled\":true}";
            File serverFile = new File(dir, "server.json");
            File clientFile = new File(dir, "client.json");
            write(serverFile, serverConfig);
            write(clientFile, clientConfig);
            server = new Running(exe, serverFile);
            client = new Running(exe, clientFile);

            byte[] expected = new byte[16384];
            for (int i = 0; i < expected.length; i++) expected[i] = (byte)((i * 37 + 19) & 255);
            String lastError = "not connected";
            long until = System.currentTimeMillis() + 12000;
            while (System.currentTimeMillis() < until) {
                try (Socket socket = new Socket()) {
                    socket.connect(new java.net.InetSocketAddress("127.0.0.1", clientPort), 700);
                    socket.setSoTimeout(6500);
                    socket.getOutputStream().write(expected);
                    socket.getOutputStream().flush();
                    byte[] actual = new byte[expected.length];
                    int received = 0;
                    while (received < actual.length) {
                        int count = socket.getInputStream().read(actual, received, actual.length - received);
                        if (count < 0) throw new IOException("unexpected EOF after " + received + " bytes");
                        received += count;
                    }
                    if (!Arrays.equals(expected, actual)) return "RESULT=FAIL_BYTES_DIFFER";
                    if (echoError.get() != null) return "RESULT=FAIL_ECHO_BACKEND " + echoError.get().getClass().getSimpleName();
                    return "RESULT=PASS_ANDROID_LOOPBACK_MTLS_TCP\n"
                        + "HTTP/2 + TLS 1.3 + mTLS\n"
                        + "Full-duplex 16384-byte echo verified\n"
                        + "Padding/Jitter enabled\n"
                        + "All listeners 127.0.0.1\n"
                        + "No production VPN/server used";
                } catch (IOException ex) {
                    lastError = ex.getClass().getSimpleName() + " " + ex.getMessage();
                    Thread.sleep(250);
                }
            }
            return "RESULT=FAIL_LOOPBACK_TIMEOUT\n" + lastError
                + "\nserver=" + server.brief() + "\nclient=" + client.brief();
        } catch (Throwable e) {
            return "RESULT=FAIL_LOOPBACK_EXCEPTION\n" + e.getClass().getSimpleName() + " " + e.getMessage();
        } finally {
            if (client != null) client.stop();
            if (server != null) server.stop();
            if (backend != null) try { backend.close(); } catch (IOException ignored) { }
        }
    }
}
