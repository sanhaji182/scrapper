"use client";

import { FormEvent, useMemo, useState, useSyncExternalStore, useTransition } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { formatRupiahShort, marketplaceClass, runKeyword } from "@/lib/format";

const RECENT_KEY = "pricescope-recent-searches";
const SORT_OPTIONS = [
  { value: "relevancy", label: "Paling relevan" },
  { value: "price_asc", label: "Harga termurah" },
  { value: "price_desc", label: "Harga tertinggi" },
  { value: "latest", label: "Terbaru" },
];
const MAX_ITEMS_OPTIONS = [10, 20, 30, 50, 100];
const MARKETPLACES = [
  { id: "tokopedia", label: "Tokopedia", helper: "Paling siap", readiness: "Langsung cari", tone: "Stabil untuk search publik dan cocok untuk pencarian cepat tanpa konfigurasi.", steps: ["Masukkan keyword", "Pilih jumlah produk", "Klik Cari"], needsSettings: false },
  { id: "blibli", label: "Blibli", helper: "Biasanya langsung jalan", readiness: "Cookie opsional", tone: "Coba tanpa cookie dulu. Jika request sesekali 403, ulangi atau gunakan proxy.", steps: ["Mulai tanpa cookie", "Gunakan 30–100 item", "Proxy jika 403"], needsSettings: false },
  { id: "shopee", label: "Shopee", helper: "Sering butuh cookie", readiness: "Cookie disarankan", tone: "Cookie browser membantu worker melewati pembatasan sesi anonymous Shopee.", steps: ["Buka Shopee", "Copy header Cookie", "Paste di Pengaturan"], needsSettings: true },
  { id: "lazada", label: "Lazada", helper: "Paling ketat", readiness: "Session + proxy ideal", tone: "Lazada sering memunculkan captcha. Cookie browser dan residential proxy paling stabil.", steps: ["Selesaikan captcha", "Copy Cookie catalog", "Gunakan proxy jika perlu"], needsSettings: true },
];
const TRUST_METRICS = [["4", "marketplace"], ["100", "produk/job"], ["AI", "normalizer"]];
const emptyRecentSearches: string[] = [];
let recentSnapshotRaw = "[]";
let recentSnapshotValue: string[] = emptyRecentSearches;

function getServerRecentSearches() { return emptyRecentSearches; }
function readRecentSearches() {
  if (typeof window === "undefined") return emptyRecentSearches;
  const raw = window.localStorage.getItem(RECENT_KEY) ?? "[]";
  if (raw === recentSnapshotRaw) return recentSnapshotValue;
  recentSnapshotRaw = raw;
  try {
    const parsed = JSON.parse(raw);
    recentSnapshotValue = Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === "string") : [];
  } catch { recentSnapshotValue = emptyRecentSearches; }
  return recentSnapshotValue;
}
function subscribeRecentSearches(callback: () => void) {
  window.addEventListener("storage", callback);
  window.addEventListener(RECENT_KEY, callback);
  return () => { window.removeEventListener("storage", callback); window.removeEventListener(RECENT_KEY, callback); };
}
function parsePrice(value: string) { return Number(value.replace(/\D/g, "")) || 0; }

type MarketplaceItem = (typeof MARKETPLACES)[number];

export function SearchHub() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [minPrice, setMinPrice] = useState("");
  const [maxPrice, setMaxPrice] = useState("");
  const [sortBy, setSortBy] = useState("relevancy");
  const [maxItems, setMaxItems] = useState(30);
  const [marketplace, setMarketplace] = useState("tokopedia");
  const recent = useSyncExternalStore(subscribeRecentSearches, readRecentSearches, getServerRecentSearches);
  const [showFilters, setShowFilters] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const selectedMarketplace = useMemo(() => MARKETPLACES.find((item) => item.id === marketplace) ?? MARKETPLACES[0], [marketplace]);
  const pricePreview = useMemo(() => {
    const min = parsePrice(minPrice); const max = parsePrice(maxPrice);
    if (!min && !max) return "semua harga";
    if (min && max) return `${formatRupiahShort(min)} – ${formatRupiahShort(max)}`;
    if (min) return `mulai ${formatRupiahShort(min)}`;
    return `hingga ${formatRupiahShort(max)}`;
  }, [maxPrice, minPrice]);

  function rememberSearch(value: string) {
    const next = [value, ...recent.filter((item) => item.toLowerCase() !== value.toLowerCase())].slice(0, 8);
    window.localStorage.setItem(RECENT_KEY, JSON.stringify(next));
    window.dispatchEvent(new Event(RECENT_KEY));
  }

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const cleanKeyword = keyword.trim();
    if (!cleanKeyword) { setError("Masukkan nama produk dulu."); return; }
    setError(null);
    startTransition(async () => {
      try {
        const history = await api.getRuns(50, 0).catch(() => null);
        const existing = history?.runs.find((run) => run.marketplace === marketplace && run.status === "SUCCEEDED" && run.item_count > 0 && runKeyword(run).trim().toLowerCase() === cleanKeyword.toLowerCase());
        if (existing) { rememberSearch(cleanKeyword); router.push(detailHref(existing.id, cleanKeyword, marketplace)); return; }
        const res = await api.submitSearch({ keyword: cleanKeyword, max_items: maxItems, sort_by: sortBy, min_price: parsePrice(minPrice), max_price: parsePrice(maxPrice) }, marketplace);
        rememberSearch(cleanKeyword);
        await waitForRunReadable(res.run_id);
        router.push(detailHref(res.run_id, cleanKeyword, marketplace));
      } catch (err) { setError(err instanceof Error ? err.message : "Gagal memulai pencarian."); }
    });
  }

  return (
    <div className="shell-container" style={{ display: "grid", gap: 18 }}>
      <section className="hero-grid" style={{ alignItems: "stretch" }}>
        <div className="soft-card hero-panel" style={{ padding: "clamp(22px, 4vw, 46px)", display: "grid", gap: 24, alignContent: "space-between", overflow: "hidden", position: "relative" }}>
          <div style={{ position: "absolute", inset: "-18% auto auto -12%", width: 360, height: 360, borderRadius: "50%", background: "color-mix(in oklab, var(--accent) 16%, transparent)", filter: "blur(18px)", pointerEvents: "none" }} />
          <div style={{ display: "grid", gap: 18, position: "relative" }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "start", flexWrap: "wrap" }}>
              <span className="hero-eyebrow"><span style={{ width: 8, height: 8, borderRadius: "50%", background: "var(--accent)", display: "inline-block" }} /> Marketplace Intelligence</span>
              <Link href="/pengaturan" className="ghost-button" style={{ padding: "9px 12px", fontSize: 12, fontWeight: 850 }}>Cookie & AI settings</Link>
            </div>
            <div>
              <h1 className="hero-title" style={{ maxWidth: 920 }}>Cari produk, bandingkan harga, ambil keputusan lebih cepat.</h1>
              <p className="hero-copy" style={{ marginTop: 18 }}>PriceScope menyatukan scraping marketplace, normalisasi produk, dan insight AI dalam workflow yang rapi untuk riset harga Indonesia.</p>
            </div>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 10, maxWidth: 560 }} className="mobile-stack">
              {TRUST_METRICS.map(([value, label]) => <MetricCard key={label} value={value} label={label} />)}
            </div>
          </div>

          <form onSubmit={onSubmit} className="visual-card" style={{ padding: 14, display: "grid", gap: 13, position: "relative", borderRadius: 30 }}>
            <div style={{ display: "grid", gap: 8 }}>
              <span className="text-faint" style={{ fontSize: 11, fontWeight: 900, letterSpacing: ".12em", textTransform: "uppercase" }}>1. Pilih sumber data</span>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(4, minmax(0, 1fr))", gap: 8 }} className="mobile-stack">
                {MARKETPLACES.map((item) => {
                  const active = marketplace === item.id;
                  return <button key={item.id} type="button" className={active ? "primary-button" : "ghost-button"} onClick={() => setMarketplace(item.id)} aria-pressed={active} style={{ padding: "12px 13px", justifyContent: "flex-start", textAlign: "left", minHeight: 78 }}><span style={{ display: "grid", gap: 3 }}><strong>{item.label}</strong><small style={{ opacity: .76, fontWeight: 750 }}>{item.helper}</small><small style={{ opacity: active ? .9 : .58, fontWeight: 700 }}>{item.readiness}</small></span></button>;
                })}
              </div>
            </div>
            <MarketplaceReadiness selected={selectedMarketplace} />
            <div style={{ display: "grid", gap: 8 }}>
              <span className="text-faint" style={{ fontSize: 11, fontWeight: 900, letterSpacing: ".12em", textTransform: "uppercase" }}>2. Masukkan produk</span>
              <div style={{ display: "grid", gridTemplateColumns: "1fr auto", gap: 10 }} className="mobile-stack">
                <input aria-label="Nama produk" aria-describedby={error ? "search-error" : undefined} className="field" value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="Cari iPhone 15, RTX 5060, Kindle 11..." style={{ padding: "18px 18px", fontSize: 16, borderRadius: 20 }} />
                <button className="primary-button" disabled={isPending} aria-busy={isPending} style={{ padding: "0 22px", minHeight: 58 }}>{isPending ? "Scanning..." : `Cari di ${selectedMarketplace.label}`}</button>
              </div>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
              <button type="button" className="ghost-button" onClick={() => setShowFilters((value) => !value)} aria-expanded={showFilters} aria-controls="search-filters" style={{ padding: "9px 12px", fontSize: 12, fontWeight: 800 }}>{showFilters ? "Tutup filter" : "3. Filter pintar"}</button>
              <span className="text-faint" style={{ fontSize: 12 }}>{pricePreview} · {maxItems} produk · {SORT_OPTIONS.find((item) => item.value === sortBy)?.label}</span>
            </div>
            {showFilters ? <div id="search-filters" className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 10 }}><input aria-label="Harga minimum" className="field" value={minPrice} onChange={(event) => setMinPrice(event.target.value)} placeholder="Min price" style={{ padding: "11px 12px" }} /><input aria-label="Harga maksimum" className="field" value={maxPrice} onChange={(event) => setMaxPrice(event.target.value)} placeholder="Max price" style={{ padding: "11px 12px" }} /><select aria-label="Urutan hasil" className="field" value={sortBy} onChange={(event) => setSortBy(event.target.value)} style={{ padding: "11px 12px" }}>{SORT_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select><select aria-label="Jumlah produk" className="field" value={maxItems} onChange={(event) => setMaxItems(Number(event.target.value))} style={{ padding: "11px 12px" }}>{MAX_ITEMS_OPTIONS.map((option) => <option key={option} value={option}>{option} produk</option>)}</select></div> : null}
            {error ? <p id="search-error" role="alert" style={{ color: "var(--error)", fontSize: 13, margin: 0, fontWeight: 760 }}>{error}</p> : null}
          </form>
        </div>

        <aside className="hero-panel" style={{ padding: 20, background: "linear-gradient(145deg, var(--ink), color-mix(in oklab, var(--accent) 24%, var(--ink)))", color: "#f6f8ef", display: "grid", alignContent: "space-between", gap: 20, overflow: "hidden", position: "relative" }}>
          <div style={{ position: "absolute", inset: "-20% -30% auto auto", width: 260, height: 260, borderRadius: "50%", background: "rgba(220,239,199,.14)", filter: "blur(8px)" }} />
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", position: "relative" }}><span style={{ fontSize: 13, opacity: .72 }}>Live decision board</span><span className="metric-pill" style={{ color: "#11140f", background: "#dcefc7" }}>AI Ready</span></div>
          <div className="visual-card" style={{ padding: 18, color: "var(--text)", animation: "floaty 5s ease-in-out infinite", position: "relative" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start", gap: 12 }}><div><p style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 6 }}>Riset cepat</p><h2 style={{ fontSize: 28, lineHeight: 1, fontWeight: 950, letterSpacing: "-.055em" }}>{keyword.trim() || "iPhone 15 128GB"}</h2></div><span className={`marketplace-badge ${marketplaceClass(selectedMarketplace.id)}`}>{selectedMarketplace.label}</span></div>
            <div style={{ marginTop: 24, display: "grid", gap: 10 }}>{[["Status sumber", selectedMarketplace.readiness], ["Scope pencarian", `${maxItems} produk`], ["Filter harga", pricePreview]].map(([label, value], index) => <div key={label} style={{ display: "flex", alignItems: "center", justifyContent: "space-between", borderTop: index ? "1px solid var(--border)" : 0, paddingTop: index ? 10 : 0, gap: 12 }}><span style={{ fontSize: 13, color: "var(--text-muted)" }}>{label}</span><strong style={{ textAlign: "right" }}>{value}</strong></div>)}</div>
          </div>
          <div style={{ display: "grid", gap: 10, position: "relative" }}>{[["01", "Search", "Submit job ke API"], ["02", "Worker", "Scrape marketplace"], ["03", "AI", "Group & summarize"]].map(([number, title, copy]) => <div key={title} style={{ display: "grid", gridTemplateColumns: "44px 1fr", gap: 10, alignItems: "center", padding: 12, border: "1px solid rgba(255,255,255,.12)", borderRadius: 20, background: "rgba(255,255,255,.06)" }}><span style={{ width: 38, height: 38, borderRadius: 14, background: "rgba(220,239,199,.14)", display: "inline-flex", alignItems: "center", justifyContent: "center", fontSize: 12, fontWeight: 950, color: "#dcefc7" }}>{number}</span><span style={{ display: "grid", gap: 2 }}><strong>{title}</strong><small style={{ opacity: .68 }}>{copy}</small></span></div>)}</div>
        </aside>
      </section>

      <section style={{ display: "grid", gridTemplateColumns: "1.2fr .8fr", gap: 14 }} className="mobile-stack">
        <div className="soft-card" style={{ borderRadius: 28, padding: 16, display: "grid", gap: 12 }}><div style={{ display: "flex", justifyContent: "space-between", gap: 10, alignItems: "center", flexWrap: "wrap" }}><strong>Marketplace playbook</strong><span className="text-faint" style={{ fontSize: 12 }}>Apa yang perlu disiapkan sebelum mencari</span></div><div style={{ display: "grid", gridTemplateColumns: "repeat(4, minmax(0, 1fr))", gap: 10 }} className="mobile-stack">{MARKETPLACES.map((item) => <PlaybookCard key={item.id} item={item} />)}</div></div>
        {recent.length > 0 ? <section className="soft-card" style={{ borderRadius: 28, padding: 16, display: "grid", alignContent: "start", gap: 12 }}><span className="text-faint" style={{ fontSize: 12, fontWeight: 850, letterSpacing: ".1em", textTransform: "uppercase" }}>Recent searches</span><div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>{recent.map((item) => <button key={item} type="button" className="ghost-button" onClick={() => setKeyword(item)} style={{ padding: "8px 14px", borderRadius: "var(--radius-xl)", fontSize: 13 }}>{item}</button>)}</div></section> : <section className="soft-card" style={{ borderRadius: 28, padding: 16, display: "grid", alignContent: "center", gap: 8 }}><span className="text-faint" style={{ fontSize: 12, fontWeight: 850, letterSpacing: ".1em", textTransform: "uppercase" }}>Getting started</span><strong>Mulai dari Tokopedia atau Blibli.</strong><p className="text-muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.5 }}>Untuk Shopee dan Lazada, buka Pengaturan jika butuh cookie browser.</p></section>}
      </section>
    </div>
  );
}

function MetricCard({ value, label }: { value: string; label: string }) { return <div className="visual-card" style={{ padding: 14 }}><strong style={{ fontSize: 24, letterSpacing: "-.05em" }}>{value}</strong><p className="text-faint" style={{ margin: "3px 0 0", fontSize: 11, fontWeight: 850, textTransform: "uppercase" }}>{label}</p></div>; }
function MarketplaceReadiness({ selected }: { selected: MarketplaceItem }) { return <div className="visual-card" style={{ padding: 14, display: "grid", gap: 12, background: "linear-gradient(135deg, color-mix(in oklab, var(--accent) 10%, var(--surface)), var(--surface))" }}><div style={{ display: "flex", justifyContent: "space-between", gap: 10, alignItems: "center", flexWrap: "wrap" }}><div><span className={`marketplace-badge ${marketplaceClass(selected.id)}`}>{selected.label}</span><h3 style={{ margin: "8px 0 3px", fontSize: 18, letterSpacing: "-.03em" }}>{selected.readiness}</h3><p className="text-muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.45 }}>{selected.tone}</p></div>{selected.needsSettings ? <Link href="/pengaturan" className="primary-button" style={{ padding: "9px 13px", minHeight: "auto", textDecoration: "none" }}>Isi cookie</Link> : <span className="marketplace-badge mp-default">Tanpa setup</span>}</div><div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 8 }} className="mobile-stack">{selected.steps.map((step, index) => <div key={step} style={{ border: "1px solid var(--border)", borderRadius: 16, padding: 11, background: "color-mix(in oklab, var(--surface) 72%, transparent)" }}><span className="text-faint" style={{ fontSize: 11, fontWeight: 900 }}>STEP {index + 1}</span><p style={{ margin: "5px 0 0", fontSize: 13, lineHeight: 1.35, fontWeight: 760 }}>{step}</p></div>)}</div></div>; }
function PlaybookCard({ item }: { item: MarketplaceItem }) { return <button type="button" className="visual-card" style={{ padding: 13, textAlign: "left", cursor: "default" }}><span className={`marketplace-badge ${marketplaceClass(item.id)}`}>{item.label}</span><h3 style={{ margin: "10px 0 5px", fontSize: 16 }}>{item.readiness}</h3><p className="text-muted" style={{ margin: 0, fontSize: 12, lineHeight: 1.45 }}>{item.needsSettings ? "Siapkan cookie di Pengaturan untuk hasil lebih stabil." : "Bisa dipakai langsung untuk pencarian awal."}</p></button>; }
function detailHref(id: string, keyword: string, marketplace: string) { const params = new URLSearchParams({ q: keyword, mp: marketplace }); return `/pencarian/${id}?${params.toString()}`; }
async function waitForRunReadable(id: string) { for (let attempt = 0; attempt < 30; attempt++) { const run = await api.getRun(id).catch(() => null); if (run?.status === "SUCCEEDED" || run?.status === "FAILED") return; await new Promise((resolve) => setTimeout(resolve, attempt < 3 ? 700 : 1200)); } }
