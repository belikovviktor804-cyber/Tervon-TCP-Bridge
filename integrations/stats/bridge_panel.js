/* Two-node Bridge panel; uses public read-only same-origin API.
 * Include only after bridge_panel.html; no credentials or private addresses.
 */
(() => {
  "use strict";
  const api = "/bridge/api/status";
  const ids = [
    "tb-sessions-active", "tb-sessions-counts", "tb-bytes",
    "tb-runtime", "tb-adaptation",
    "tb-padding-bytes", "tb-padding-overhead", "tb-padding-direction",
    "tb-padding-activity", "tb-jitter-delays", "tb-errors",
  ];
  const byId = (id) => document.getElementById(id);
  const setText = (id, value) => {
    const element = byId(id);
    if (element) element.textContent = String(value);
  };
  const integer = (value) =>
    typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
  const number = (value) =>
    typeof value === "number" && Number.isFinite(value) && value >= 0;
  const fmt = (value) => integer(value) ? value.toLocaleString("ru-RU") : "—";
  const decimal = (value) => number(value) ? value.toFixed(1) : "—";
  const bytes = (value) => {
    if (!number(value)) return "—";
    const units = ["Б", "КиБ", "МиБ", "ГиБ", "ТиБ", "ПиБ"];
    let n = value;
    let index = 0;
    while (n >= 1024 && index < units.length - 1) {
      n /= 1024;
      index++;
    }
    return n.toFixed(index === 0 ? 0 : 1) + " " + units[index];
  };
  const state = (value) => ({
    NOT_IMPLEMENTED: "Не реализовано",
    OFF: "Выкл.",
    ACTIVE: "Активно",
    ENABLED: "Включено",
    ERROR: "Ошибка",
  })[value] || "—";
  const clear = () => {
    ids.forEach((id) => setText(id, "—"));
    setText("tb-updated", "—");
  };
  const render = (data) => {
    const node = data && data.schema === "tervon.bridge.dashboard.v1" &&
      data.nodes && data.nodes.homeserver;
    if (!node || node.status !== "ok" || !node.metrics) {
      clear();
      setText("tb-status", "Метрики Homeserver недоступны");
      return;
    }
    const m = node.metrics;
    setText("tb-status", m.ready === true ? "Bridge готов" : "Bridge не готов");
    setText("tb-sessions-active", fmt(m.sessions_active));
    setText("tb-sessions-counts",
      [m.sessions_started, m.sessions_completed, m.sessions_rejected].map(fmt).join(" / "));
    setText("tb-bytes", bytes(m.upload_bytes) + " / " + bytes(m.download_bytes));
    setText("tb-runtime", decimal(m.uptime_seconds) + " с / " +
      fmt(m.goroutines) + " / " + bytes(m.memory_alloc_bytes));
    // OFF is an activity state; legacy binaries may not report configuration.
    const inactivePadding = (value) => value !== "OFF" ? state(value) :
      m.padding_configured === true ? "Ожидает подходящий поток" :
      m.padding_configured === false ? "Выключено" :
      "Нет активности (настройка неизвестна)";
    const paddingStatus = inactivePadding(m.padding_state);
    const jitterStatus = m.jitter_configured === true &&
      m.jitter_state === "OFF" ? "Ожидает поток" : state(m.jitter_state);
    setText("tb-adaptation", paddingStatus + " / " + jitterStatus);
    setText("tb-padding-bytes", bytes(m.padding_upload_bytes) + " / " +
      bytes(m.padding_download_bytes));
    const sideState = (v) => v === "OFF" ?
      (m.padding_configured === true ? "Ожидание" :
        m.padding_configured === false ? "Выключено" : "Нет активности") :
      state(v);
    setText("tb-padding-direction", sideState(m.padding_upload_state) +
      " / " + sideState(m.padding_download_state));
    setText("tb-padding-activity",
      fmt(m.padding_upload_frames) + " / " + fmt(m.padding_download_frames) +
      " кадров; " + fmt(m.padding_upload_transitions) + " / " +
      fmt(m.padding_download_transitions) + " переходов");
    setText("tb-padding-overhead",
      [m.padding_upload_wire_overhead_percent,
        m.padding_download_wire_overhead_percent].map(
          (value) => number(value) ? value.toFixed(2) + "%" : "—"
        ).join(" / "));

    const jitterSide = (frames, delayNs) => integer(frames) &&
      number(delayNs) ? fmt(frames) + " / " + (delayNs / 1e6).toFixed(2) + " мс" : "—";
    setText("tb-jitter-delays", jitterSide(m.jitter_upload_frames, m.jitter_upload_delay_ns) +
      " / " + jitterSide(m.jitter_download_frames, m.jitter_download_delay_ns));

    const errors = m.errors && typeof m.errors === "object" ?
      Object.values(m.errors) : [];
    setText("tb-errors", errors.length && errors.every(integer) ?
      fmt(errors.reduce((sum, v) => sum + v, 0)) : "—");
    const at = typeof data.generated_at === "string" ? new Date(data.generated_at) : null;
    setText("tb-updated", at && !Number.isNaN(at.getTime()) ?
      at.toLocaleString("ru-RU") : "—");
  };
  // Finserver is optional. Never interpret a missing/stale tunnel as zero
  // traffic, and never render untrusted HTML or expose remote endpoint URLs.
  const finIds = ["tb-fin-active", "tb-fin-bytes", "tb-fin-pad",
    "tb-fin-overhead", "tb-fin-jitter", "tb-fin-adaptation",
    "tb-fin-backend", "tb-fin-latency", "tb-fin-tls"];
  const renderFin = (data) => {
    const node = data && data.schema === "tervon.bridge.dashboard.v1" &&
      data.nodes && data.nodes.finserver;
    if (!node || node.status !== "ok" || !node.metrics) {
      finIds.forEach((id) => setText(id, "—"));
      setText("tb-fin-status", node ? "Finserver: метрики недоступны" :
        "Finserver: источник не подключён");
      return;
    }
    const m = node.metrics;
    setText("tb-fin-status", m.ready === true ?
      "Finserver готов" : "Finserver не готов");
    setText("tb-fin-active", fmt(m.sessions_active));
    setText("tb-fin-bytes", bytes(m.upload_bytes) + " / " + bytes(m.download_bytes));
    setText("tb-fin-pad", fmt(m.padding_upload_frames) + " / " +
      fmt(m.padding_download_frames));
    setText("tb-fin-overhead", [
      m.padding_upload_wire_overhead_percent,
      m.padding_download_wire_overhead_percent,
    ].map((v) => number(v) ? v.toFixed(2) + "%" : "—").join(" / "));
    const delayed = (frames, ns) => integer(frames) && number(ns) ?
      fmt(frames) + " / " + (ns / 1e6).toFixed(2) + " мс" : "—";
    setText("tb-fin-jitter",
      delayed(m.jitter_upload_frames, m.jitter_upload_delay_ns) +
      " / " + delayed(m.jitter_download_frames, m.jitter_download_delay_ns));
    const padding = m.padding_state === "OFF" &&
      m.padding_configured === true ? "Ожидание" : state(m.padding_state);
    setText("tb-fin-adaptation", padding + " / " + state(m.jitter_state));
    setText("tb-fin-backend",
      fmt(m.backend_dial_success) + " / " + fmt(m.backend_dial_failure));
    // These are measurements made by the Decapsulator, not Homeserver.
    // Show a factual sample count when no latency observations exist.
    if (integer(m.connect_latency_samples) && m.connect_latency_samples > 0) {
      setText("tb-fin-latency", [
        m.connect_latency_p50_ms, m.connect_latency_p95_ms,
        m.connect_latency_p99_ms,
      ].map(decimal).join(" / ") + " мс (" + fmt(m.connect_latency_samples) + ")");
    } else {
      setText("tb-fin-latency",
        integer(m.connect_latency_samples) ? "0 замеров" : "—");
    }
    setText("tb-fin-tls", fmt(m.tls13_h2_mtls_requests));
  };
  const refresh = async () => {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);
    try {
      const response = await fetch(api, {
        method: "GET", credentials: "same-origin", cache: "no-store",
        redirect: "error", signal: controller.signal,
        headers: { Accept: "application/json" },
      });
      if (!response.ok) throw new Error("unavailable");
      const result = await response.json();
      render(result);
      renderFin(result);
    } catch (_) {
      render(null);
      renderFin(null);
    } finally {
      clearTimeout(timeout);
    }
  };
  const init = () => {
    if (!byId("tb-panel")) return;
    refresh();
    setInterval(refresh, 15000);
  };
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init, { once: true });
  } else {
    init();
  }
})();
