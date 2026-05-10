"use client";

import { FormEvent, useMemo, useState, useTransition } from "react";
import { api, type AISettings } from "@/lib/api";

const PROVIDERS = [
  { id: "openai", name: "OpenAI", helper: "Paling umum untuk GPT models", models: ["gpt-4.1-mini", "gpt-4.1", "o4-mini"] },
  { id: "groq", name: "Groq", helper: "Cepat dan murah untuk Llama", models: ["llama-3.3-70b-versatile", "mixtral-8x7b-32768"] },
  { id: "gemini", name: "Gemini", helper: "Google via OpenAI-compatible API", models: ["gemini-2.0-flash", "gemini-1.5-pro"] },
  { id: "ollama", name: "Ollama Lokal", helper: "Tanpa API key, jalan di komputer/server sendiri", models: ["llama3.2", "qwen2.5", "gemma3"] },
];

const DEFAULT_SETTINGS: AISettings = {
  provider: "openai",
  api_key: "",
  model: "gpt-4.1-mini",
  timeout_sec: 30,
  max_retries: 2,
  configured: false,
};

export function AISettingsPanel({ initialSettings }: { initialSettings: AISettings | null }) {
  const [settings, setSettings] = useState<AISettings>(initialSettings ?? DEFAULT_SETTINGS);
  const [apiKey, setApiKey] = useState(initialSettings?.api_key ?? "");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, startSaving] = useTransition();
  const [isTesting, startTesting] = useTransition();

  const provider = useMemo(() => PROVIDERS.find((item) => item.id === settings.provider) ?? PROVIDERS[0], [settings.provider]);
  const needsKey = settings.provider !== "ollama";

  function updateProvider(providerID: string) {
    const nextProvider = PROVIDERS.find((item) => item.id === providerID) ?? PROVIDERS[0];
    setSettings((current) => ({ ...current, provider: nextProvider.id, model: nextProvider.models[0] }));
  }

  function onSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage(null);
    setError(null);
    startSaving(async () => {
      try {
        const next = await api.updateAISettings({
          provider: settings.provider,
          api_key: apiKey,
          model: settings.model,
          timeout_sec: settings.timeout_sec,
          max_retries: settings.max_retries,
        });
        setSettings(next);
        setApiKey(next.api_key ?? "");
        setMessage("Pengaturan AI tersimpan. Normalisasi dan rekomendasi berikutnya akan memakai model ini.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menyimpan pengaturan AI.");
      }
    });
  }

  function onTest() {
    setMessage(null);
    setError(null);
    startTesting(async () => {
      try {
        await api.testAISettings();
        setMessage("Koneksi AI berhasil. Model siap dipakai.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Test koneksi gagal.");
      }
    });
  }

  return (
    <div className="shell-container" style={{ display: "grid", gap: 20 }}>
      <header className="soft-card" style={{ borderRadius: 34, padding: "clamp(22px, 4vw, 42px)" }}>
        <span className="hero-eyebrow">✦ AI Control Room</span>
        <h1 style={{ margin: "18px 0 10px", fontSize: "clamp(34px, 6vw, 72px)", lineHeight: .9, letterSpacing: "-.08em", fontWeight: 950 }}>Atur AI tanpa buka terminal.</h1>
        <p className="hero-copy">Pilih provider, tempel API key, pilih model, lalu test koneksi. Pengaturan ini langsung dipakai oleh tab rekomendasi AI.</p>
      </header>

      <form onSubmit={onSave} className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "0.9fr 1.1fr", gap: 16 }}>
        <section className="soft-card" style={{ borderRadius: 28, padding: 16, display: "grid", gap: 10 }}>
          <p className="sidebar-section-label" style={{ padding: 0 }}>Pilih provider</p>
          {PROVIDERS.map((item) => {
            const active = item.id === settings.provider;
            return (
              <button key={item.id} type="button" onClick={() => updateProvider(item.id)} aria-pressed={active} className={`nav-item ${active ? "active" : ""}`} style={{ width: "100%", border: 0, textAlign: "left", background: active ? undefined : "transparent" }}>
                <span className="nav-icon" aria-hidden="true">{item.id === "ollama" ? "◌" : "✦"}</span>
                <span className="nav-copy"><span>{item.name}</span><small>{item.helper}</small></span>
              </button>
            );
          })}
        </section>

        <section className="soft-card" style={{ borderRadius: 28, padding: 20, display: "grid", gap: 16 }}>
          <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
            <div>
              <h2 style={{ margin: 0, fontSize: 24, fontWeight: 920, letterSpacing: "-.04em" }}>{provider.name}</h2>
              <p className="text-muted" style={{ margin: "4px 0 0", fontSize: 13 }}>{needsKey ? "Butuh API key dari provider." : "Tidak butuh API key, pastikan Ollama aktif."}</p>
            </div>
            <span className={`status-badge ${settings.configured ? "status-succeeded" : "status-queued"}`}><span className="dot" />{settings.configured ? "Configured" : "Needs setup"}</span>
          </div>

          <label style={{ display: "grid", gap: 8 }}>
            <span style={{ fontSize: 13, fontWeight: 800 }}>API Key {needsKey ? "" : "(opsional)"}</span>
            <input className="field" type="password" value={apiKey} onChange={(event) => setApiKey(event.target.value)} placeholder={needsKey ? "Tempel API key di sini" : "Kosongkan untuk Ollama"} style={{ padding: "13px 14px" }} />
          </label>

          <label style={{ display: "grid", gap: 8 }}>
            <span style={{ fontSize: 13, fontWeight: 800 }}>Model</span>
            <input className="field" list="ai-models" value={settings.model} onChange={(event) => setSettings((current) => ({ ...current, model: event.target.value }))} style={{ padding: "13px 14px" }} />
            <datalist id="ai-models">{provider.models.map((model) => <option key={model} value={model} />)}</datalist>
          </label>

          <div className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
            <label style={{ display: "grid", gap: 8 }}><span style={{ fontSize: 13, fontWeight: 800 }}>Timeout detik</span><input className="field" type="number" min={5} max={180} value={settings.timeout_sec} onChange={(event) => setSettings((current) => ({ ...current, timeout_sec: Number(event.target.value) }))} style={{ padding: "13px 14px" }} /></label>
            <label style={{ display: "grid", gap: 8 }}><span style={{ fontSize: 13, fontWeight: 800 }}>Max retry</span><input className="field" type="number" min={0} max={5} value={settings.max_retries} onChange={(event) => setSettings((current) => ({ ...current, max_retries: Number(event.target.value) }))} style={{ padding: "13px 14px" }} /></label>
          </div>

          {message ? <p role="status" className="soft-panel" style={{ margin: 0, borderRadius: 16, padding: 12, color: "var(--accent-strong)", fontSize: 13 }}>{message}</p> : null}
          {error ? <p role="alert" className="soft-panel" style={{ margin: 0, borderRadius: 16, padding: 12, color: "var(--error)", fontSize: 13 }}>{error}</p> : null}

          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            <button className="primary-button" disabled={isSaving} aria-busy={isSaving} style={{ padding: "12px 18px" }}>{isSaving ? "Menyimpan..." : "Simpan pengaturan"}</button>
            <button type="button" className="ghost-button" disabled={isTesting} aria-busy={isTesting} onClick={onTest} style={{ padding: "12px 18px", fontWeight: 800 }}>{isTesting ? "Testing..." : "Test koneksi"}</button>
          </div>
        </section>
      </form>
    </div>
  );
}
