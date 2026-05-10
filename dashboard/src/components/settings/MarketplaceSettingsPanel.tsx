"use client";

import { FormEvent, useMemo, useState, useTransition } from "react";
import { api, type MarketplaceSettings } from "@/lib/api";

const MARKETPLACE_GUIDES = [
  {
    id: "tokopedia",
    name: "Tokopedia",
    badge: "Siap langsung",
    status: "Tidak perlu cookie",
    accent: "mp-tokopedia",
    description: "Marketplace paling stabil untuk search publik. Mulai dari sini kalau ingin hasil cepat tanpa konfigurasi tambahan.",
    steps: ["Pilih Tokopedia di halaman pencarian", "Masukkan keyword produk", "Klik Cari dan tunggu hasil"],
  },
  {
    id: "blibli",
    name: "Blibli",
    badge: "Cookie opsional",
    status: "Biasanya tanpa setup",
    accent: "mp-blibli",
    description: "Blibli biasanya bisa langsung dipakai. Jika sesekali 403, ulangi pencarian atau gunakan proxy residential.",
    steps: ["Coba cari tanpa cookie dulu", "Gunakan 30–100 produk", "Jika 403 berulang, aktifkan proxy"],
  },
  {
    id: "shopee",
    name: "Shopee",
    badge: "Butuh sesi",
    status: "Cookie disarankan",
    accent: "mp-shopee",
    description: "Shopee sering membatasi request anonymous. Cookie browser membantu worker terlihat seperti sesi browser yang valid.",
    steps: ["Buka Shopee di browser", "Cari produk dan copy header Cookie", "Paste ke field Shopee di bawah"],
  },
  {
    id: "lazada",
    name: "Lazada",
    badge: "Paling ketat",
    status: "Cookie + proxy ideal",
    accent: "mp-lazada",
    description: "Lazada sering menampilkan captcha. Cookie browser yang sudah lolos captcha dan residential/mobile proxy memberi peluang sukses terbaik.",
    steps: ["Buka Lazada dan selesaikan captcha", "Copy Cookie dari request catalog", "Paste ke field Lazada di bawah"],
  },
];

export function MarketplaceSettingsPanel({ initialSettings }: { initialSettings: MarketplaceSettings | null }) {
  const [activeGuide, setActiveGuide] = useState("shopee");
  const [shopeeCookie, setShopeeCookie] = useState(initialSettings?.shopee_cookie_header ?? "");
  const [lazadaCookie, setLazadaCookie] = useState(initialSettings?.lazada_cookie_header ?? "");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, startSaving] = useTransition();

  const selectedGuide = useMemo(() => MARKETPLACE_GUIDES.find((item) => item.id === activeGuide) ?? MARKETPLACE_GUIDES[2], [activeGuide]);
  const shopeeReady = shopeeCookie.trim().length > 0;
  const lazadaReady = lazadaCookie.trim().length > 0;

  function onSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setMessage(null);
    startSaving(async () => {
      try {
        const saved = await api.updateMarketplaceSettings({
          shopee_cookie_header: shopeeCookie,
          lazada_cookie_header: lazadaCookie,
        });
        setShopeeCookie(saved.shopee_cookie_header ?? "");
        setLazadaCookie(saved.lazada_cookie_header ?? "");
        setMessage("Cookie runtime tersimpan. Job berikutnya akan otomatis memakai cookie yang sesuai marketplace.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menyimpan cookie marketplace.");
      }
    });
  }

  return (
    <section className="shell-container" style={{ display: "grid", gap: 18, marginTop: 18 }}>
      <div className="soft-card" style={{ borderRadius: 34, padding: "clamp(18px, 3vw, 30px)", display: "grid", gap: 18, overflow: "hidden", position: "relative" }}>
        <div style={{ position: "absolute", inset: "auto -10% -35% auto", width: 280, height: 280, borderRadius: "50%", background: "color-mix(in oklab, var(--accent) 16%, transparent)", filter: "blur(12px)", pointerEvents: "none" }} />

        <div className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "1.05fr .95fr", gap: 18, position: "relative" }}>
          <div style={{ display: "grid", gap: 14, alignContent: "start" }}>
            <div>
              <span className="hero-eyebrow">Marketplace Access</span>
              <h2 style={{ margin: "12px 0 8px", fontSize: "clamp(30px, 4vw, 54px)", lineHeight: .95, letterSpacing: "-.075em", fontWeight: 950 }}>Siapkan sesi marketplace tanpa edit env.</h2>
              <p className="hero-copy" style={{ margin: 0, fontSize: 16 }}>Cookie dimasukkan lewat dashboard, disimpan hanya di runtime API, lalu ikut dikirim ke worker untuk job berikutnya.</p>
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 10 }} className="mobile-stack">
              <StatusTile label="Shopee cookie" ready={shopeeReady} />
              <StatusTile label="Lazada cookie" ready={lazadaReady} />
            </div>

            <div className="visual-card" style={{ padding: 14, display: "grid", gap: 10 }}>
              <strong style={{ fontSize: 14 }}>Cara ambil cookie yang aman</strong>
              {[
                "Buka marketplace di browser yang sama dengan IP/proxy target.",
                "Cari produk dan selesaikan captcha jika muncul.",
                "DevTools → Network → klik request search/catalog → copy header Cookie.",
                "Paste di field bawah. Jangan commit cookie ke Git.",
              ].map((item, index) => (
                <div key={item} style={{ display: "grid", gridTemplateColumns: "26px 1fr", gap: 10, alignItems: "start" }}>
                  <span className="marketplace-badge mp-default" style={{ width: 26, height: 26, padding: 0, justifyContent: "center" }}>{index + 1}</span>
                  <span className="text-muted" style={{ fontSize: 13, lineHeight: 1.45 }}>{item}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="visual-card" style={{ padding: 14, display: "grid", gap: 12, alignContent: "start", background: "linear-gradient(145deg, color-mix(in oklab, var(--ink) 92%, var(--accent)), var(--ink))", color: "#f6f8ef" }}>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 8 }}>
              {MARKETPLACE_GUIDES.map((item) => {
                const active = item.id === selectedGuide.id;
                return (
                  <button key={item.id} type="button" className={active ? "primary-button" : "ghost-button"} onClick={() => setActiveGuide(item.id)} style={{ padding: "10px 11px", minHeight: 62, textAlign: "left", justifyContent: "flex-start", color: active ? undefined : "#f6f8ef", background: active ? undefined : "rgba(255,255,255,.06)", borderColor: active ? undefined : "rgba(255,255,255,.14)" }}>
                    <span style={{ display: "grid", gap: 2 }}>
                      <strong>{item.name}</strong>
                      <small style={{ opacity: .74, fontWeight: 760 }}>{item.badge}</small>
                    </span>
                  </button>
                );
              })}
            </div>

            <div style={{ border: "1px solid rgba(255,255,255,.14)", borderRadius: 24, padding: 16, background: "rgba(255,255,255,.07)" }}>
              <span className={`marketplace-badge ${selectedGuide.accent}`}>{selectedGuide.status}</span>
              <h3 style={{ margin: "12px 0 7px", fontSize: 26, lineHeight: 1, letterSpacing: "-.05em" }}>{selectedGuide.name}</h3>
              <p style={{ margin: 0, color: "rgba(246,248,239,.76)", lineHeight: 1.5, fontSize: 13 }}>{selectedGuide.description}</p>
              <div style={{ display: "grid", gap: 8, marginTop: 16 }}>
                {selectedGuide.steps.map((step, index) => (
                  <div key={step} style={{ display: "flex", gap: 9, alignItems: "center" }}>
                    <span style={{ width: 22, height: 22, borderRadius: 999, background: "rgba(220,239,199,.16)", color: "#dcefc7", display: "inline-flex", alignItems: "center", justifyContent: "center", fontSize: 12, fontWeight: 900 }}>{index + 1}</span>
                    <span style={{ fontSize: 13, fontWeight: 720 }}>{step}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <form onSubmit={onSave} className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14, position: "relative" }}>
          <CookieField title="Shopee Cookie Header" badge={shopeeReady ? "Runtime aktif" : "Kosong"} value={shopeeCookie} onChange={setShopeeCookie} onClear={() => setShopeeCookie("")} placeholder="SPC_F=...; REC_T_ID=...; csrftoken=..." />
          <CookieField title="Lazada Cookie Header" badge={lazadaReady ? "Runtime aktif" : "Kosong"} value={lazadaCookie} onChange={setLazadaCookie} onClear={() => setLazadaCookie("")} placeholder="lzd_cid=...; _tb_token_=...; ..." />

          <div style={{ gridColumn: "1 / -1", display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
            <div>
              {message ? <p style={{ color: "var(--success)", margin: 0, fontWeight: 760 }}>{message}</p> : null}
              {error ? <p style={{ color: "var(--error)", margin: 0, fontWeight: 760 }}>{error}</p> : null}
              {!message && !error ? <p className="text-muted" style={{ margin: 0, fontSize: 13 }}>Cookie tersimpan sementara. Jika API restart, isi ulang dari halaman ini.</p> : null}
            </div>
            <button className="primary-button" disabled={isSaving} style={{ padding: "12px 18px" }}>{isSaving ? "Menyimpan..." : "Simpan Cookie Runtime"}</button>
          </div>
        </form>
      </div>
    </section>
  );
}

function StatusTile({ label, ready }: { label: string; ready: boolean }) {
  return (
    <div className="visual-card" style={{ padding: 14, display: "grid", gap: 5 }}>
      <span className="text-faint" style={{ fontSize: 11, fontWeight: 900, letterSpacing: ".1em", textTransform: "uppercase" }}>{label}</span>
      <strong style={{ fontSize: 18 }}>{ready ? "Terpasang" : "Belum diisi"}</strong>
      <span className={`marketplace-badge ${ready ? "mp-tokopedia" : "mp-default"}`} style={{ justifySelf: "start" }}>{ready ? "Siap job berikutnya" : "Opsional"}</span>
    </div>
  );
}

function CookieField({ title, badge, value, onChange, onClear, placeholder }: { title: string; badge: string; value: string; onChange: (value: string) => void; onClear: () => void; placeholder: string }) {
  return (
    <label className="soft-card" style={{ borderRadius: 24, padding: 14, display: "grid", gap: 10 }}>
      <span style={{ display: "flex", justifyContent: "space-between", gap: 10, alignItems: "center" }}>
        <strong>{title}</strong>
        <span className="marketplace-badge mp-default">{badge}</span>
      </span>
      <textarea className="field" value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} rows={5} style={{ padding: 13, fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace", fontSize: 12, lineHeight: 1.45 }} />
      <button type="button" className="ghost-button" onClick={onClear} style={{ justifySelf: "start", padding: "8px 11px", fontSize: 12 }}>Bersihkan field</button>
    </label>
  );
}
