/* Homeserver-only Bridge panel; uses public read-only same-origin API.
 * Include only after bridge_panel.html; no credentials or private addresses.
 */
(() => {
  "use strict";
  const api = "/bridge/api/status";
  const ids = [
    "tb-sessions-active", "tb-sessions-counts", "tb-bytes", "tb-backend",
    "tb-latency", "tb-tls", "tb-runtime", "tb-adaptation", "tb-errors",
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
    setText("tb-backend", fmt(m.backend_dial_success) + " / " + fmt(m.backend_dial_failure));
    setText("tb-latency", [m.connect_latency_p50_ms,
      m.connect_latency_p95_ms, m.connect_latency_p99_ms].map(decimal).join(" / ") + " мс");
    setText("tb-tls", fmt(m.tls13_h2_mtls_requests));
    setText("tb-runtime", decimal(m.uptime_seconds) + " с / " +
      fmt(m.goroutines) + " / " + bytes(m.memory_alloc_bytes));
    setText("tb-adaptation", state(m.padding_state) + " / " + state(m.jitter_state));
    const errors = m.errors && typeof m.errors === "object" ?
      Object.values(m.errors) : [];
    setText("tb-errors", errors.length && errors.every(integer) ?
      fmt(errors.reduce((sum, v) => sum + v, 0)) : "—");
    const at = typeof data.generated_at === "string" ? new Date(data.generated_at) : null;
    setText("tb-updated", at && !Number.isNaN(at.getTime()) ?
      at.toLocaleString("ru-RU") : "—");
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
      render(await response.json());
    } catch (_) {
      render(null);
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
