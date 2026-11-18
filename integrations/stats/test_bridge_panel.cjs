"use strict";
const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "bridge_panel.js"), "utf8");
const tick = () => new Promise((resolve) => setImmediate(resolve));

function mount(fetcher) {
  const elements = new Map();
  for (const id of [
    "tb-panel", "tb-status", "tb-updated", "tb-sessions-active",
    "tb-sessions-counts", "tb-bytes", "tb-backend", "tb-latency",
    "tb-tls", "tb-runtime", "tb-adaptation",
    "tb-padding-bytes", "tb-padding-overhead", "tb-jitter-delays", "tb-errors",
  ]) elements.set(id, { textContent: "" });
  const calls = [];
  const context = {
    document: {
      readyState: "complete",
      getElementById: (id) => elements.get(id) || null,
    },
    fetch: async (url, options) => {
      calls.push({ url, options });
      return fetcher();
    },
    AbortController,
    setTimeout: () => 1,
    clearTimeout: () => {},
    setInterval: () => 1,
    Date,
  };
  vm.runInNewContext(source, context, { filename: "bridge_panel.js" });
  return { elements, calls };
}

function response() {
  return {
    ok: true,
    json: async () => ({
      schema: "tervon.bridge.dashboard.v1",
      generated_at: "2026-11-17T00:00:00Z",
      nodes: {
        homeserver: {
          status: "ok",
          metrics: {
            ready: true,
            sessions_active: 2,
            sessions_started: 10,
            sessions_completed: 7,
            sessions_rejected: 1,
            upload_bytes: 2048,
            download_bytes: 1024,
            backend_dial_success: 9,
            backend_dial_failure: 1,
            connect_latency_p50_ms: 1,
            connect_latency_p95_ms: 5,
            connect_latency_p99_ms: 10,
            tls13_h2_mtls_requests: 5,
            uptime_seconds: 3600,
            goroutines: 12,
            memory_alloc_bytes: 1024,
            padding_state: "NOT_IMPLEMENTED",
            jitter_state: "NOT_IMPLEMENTED",
            padding_upload_bytes: 1024,
            padding_download_bytes: 512,
            padding_upload_wire_overhead_percent: 2.5,
            padding_download_wire_overhead_percent: 1.25,
            jitter_upload_frames: 4,
            jitter_download_frames: 3,
            jitter_upload_delay_ns: 2500000,
            jitter_download_delay_ns: 1000000,
            errors: { timeout: 1, backend_dial: 2 },
          },
        },
      },
    }),
  };
}

test("renders local Homeserver metrics with fixed, same-origin API", async () => {
  const { elements, calls } = mount(async () => response());
  await tick();
  await tick();
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, "/bridge/api/status");
  assert.equal(calls[0].options.credentials, "same-origin");
  assert.equal(calls[0].options.redirect, "error");
  assert.equal(elements.get("tb-status").textContent, "Bridge готов");
  assert.equal(elements.get("tb-sessions-active").textContent, "2");
  assert.equal(elements.get("tb-bytes").textContent, "2.0 КиБ / 1.0 КиБ");
  assert.equal(elements.get("tb-adaptation").textContent, "Не реализовано / Не реализовано");
  assert.equal(elements.get("tb-padding-bytes").textContent, "1.0 КиБ / 512 Б");
  assert.equal(elements.get("tb-padding-overhead").textContent, "2.50% / 1.25%");
  assert.equal(elements.get("tb-jitter-delays").textContent, "4 / 2.50 мс / 3 / 1.00 мс");
  assert.equal(elements.get("tb-errors").textContent, "3");
});

test("configured but inactive padding is shown as waiting, not disabled", async () => {
  const previous = response();
  const original = await previous.json();
  original.nodes.homeserver.metrics.padding_configured = true;
  original.nodes.homeserver.metrics.padding_rate_configured = true;
  original.nodes.homeserver.metrics.padding_state = "OFF";
  original.nodes.homeserver.metrics.jitter_configured = true;
  original.nodes.homeserver.metrics.jitter_state = "ENABLED";
  const { elements } = mount(async () => ({
    ok: true, json: async () => original,
  }));
  await tick();
  await tick();
  assert.equal(elements.get("tb-adaptation").textContent,
    "Ожидает подходящий поток / Включено");
});

test("does not render remote node data or stale values when source is unavailable", async () => {
  const { elements } = mount(async () => ({
    ok: true, json: async () => ({
      schema: "tervon.bridge.dashboard.v1",
      nodes: { finserver: { status: "ok", metrics: { sessions_active: 100 } } },
    }),
  }));
  await tick();
  await tick();
  assert.equal(elements.get("tb-status").textContent, "Метрики Homeserver недоступны");
  assert.equal(elements.get("tb-sessions-active").textContent, "—");
});

test("failed fetch shows unavailable rather than fabricated zeroes", async () => {
  const { elements } = mount(async () => { throw new Error("connection refused"); });
  await tick();
  await tick();
  assert.equal(elements.get("tb-status").textContent, "Метрики Homeserver недоступны");
  assert.equal(elements.get("tb-errors").textContent, "—");
});

test("never uses HTML injection sinks or remote metrics endpoints", () => {
  assert.doesNotMatch(source, /\.innerHTML\s*=|insertAdjacentHTML|eval\s*\(/);
  assert.doesNotMatch(source, /finserver|https?:\/\//i);
});
