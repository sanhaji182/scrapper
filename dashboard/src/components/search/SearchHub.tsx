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
  {
    id: "tokopedia",
    label: "Tokopedia",
    helper: "Paling siap",
    readiness: "Langsung cari",
    tone: "Aman untuk mulai tanpa setup tambahan.",
    steps: ["Masukkan keyword produk", "Pilih jumlah produk", "Klik Bandingkan"],
    needsSettings: false,
  },
  {
    id: "blibli",
    label: "Blibli",
    helper: "Biasanya langsung jalan",
    readiness: "Cookie opsional",
    tone: "Kalau kena 403, coba ulang atau gunakan proxy.",
    steps: ["Mulai tanpa cookie dulu", "Gunakan 30–100 produk", "Jika gagal 403, pertimbangkan proxy"],
    needsSettings: false,
  },
  {
    id: "shopee",
    label: "Shopee",
    helper: "Sering butuh cookie",
    readiness: "Siapkan cookie jika gagal",
    tone: "Cookie browser membantu melewati 403 dan sesi anonymous yang ketat.",
    steps: ["Buka Shopee di browser", "Cari produk dan copy header Cookie", "Paste di Pengaturan → Shopee Cookie Header"],
    needsSettings: true,
  },
  {
    id: "lazada",
    label: "Lazada",
    helper: "Paling ketat",
    readiness: "Butuh session/proxy",
    tone: "Lazada sering mengirim captcha; cookie browser dan residential proxy paling stabil.",
    steps: ["Buka Lazada dan selesaikan captcha", "Copy header Cookie dari request catalog", "Paste di Pengaturan → Lazada Cookie Header"],
    needsSettings: true,
  },
];

const emptyRecentSearches: string[] = [];
let recentSnapshotRaw = "[]";
let recentSnapshotValue: string[] = emptyRecentSearches;

function getServerRecentSearches() {
  return emptyRecentSearches;
}

function readRecentSearches() {
  if (typeof window === "undefined") return emptyRecentSearches;
  const raw = window.localStorage.getItem(RECENT_KEY) ?? "[]";
  if (raw === recentSnapshotRaw) return recentSnapshotValue;
  recentSnapshotRaw = raw;
  try {
    const parsed = JSON.parse(raw);
    recentSnapshotValue = Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === "string") : [];
  } catch {
    recentSnapshotValue = emptyRecentSearches;
  }
  return recentSnapshotValue;
}

function subscribeRecentSearches(callback: () => void) {
  window.addEventListener("storage", callback);
  window.addEventListener(RECENT_KEY, callback);
  return () => {
    window.removeEventListener("storage", callback);
    window.removeEventListener(RECENT_KEY, callback);
  };
}

function parsePrice(value: string) {
  return Number(value.replace(/\D/g, "")) || 0;
}

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
    const min = parsePrice(minPrice);
    const max = parsePrice(maxPrice);
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
    if (!cleanKeyword) {
      setError("Masukkan nama produk dulu.");
      return;
    }
    setError(null);
    startTransition(async () => {
      try {
        const history = await api.getRuns(50, 0).catch(() => null);
        const existing = history?.runs.find((run) => run.marketplace === marketplace && run.status === "SUCCEEDED" && run.item_count > 0 && runKeyword(run).trim().toLowerCase() === cleanKeyword.toLowerCase());
        if (existing) {
          rememberSearch(cleanKeyword);
          router.push(detailHref(existing.id, cleanKeyword, marketplace));
          return;
        }

        const res = await api.submitSearch({
          keyword: cleanKeyword,
          max_items: maxItems,
          sort_by: sortBy,
          min_price: parsePrice(minPrice),
          max_price: parsePrice(maxPrice),
        }, marketplace);
        rememberSearch(cleanKeyword);
        await waitForRunReadable(res.run_id);
        router.push(detailHref(res.run_id, cleanKeyword, marketplace));
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal memulai pencarian.");
      }
    });
  }

  return (
    <div className="shell-container" style={{ display: "grid", gap: 20 }}>
      <section className="hero-grid">
        <div className="soft-card hero-panel" style={{ padding: "clamp(24px, 5vw, 58px)", display: "flex", flexDirection: "column", justifyContent: "space-between", gap: 34 }}>
          <div style={{ display: "grid", gap: 22, position: "relative", zIndex: 1 }}>
            <span className="hero-eyebrow"><span style={{ width: 8, height: 8, borderRadius: "50%", background: "var(--accent)", display: "inline-block" }} /> Creative price radar</span>
            <h1 className="hero-title">Temukan harga terbaik dalam satu sapuan biru.</h1>
            <p className="hero-copy">Search cepat lintas Tokopedia, Shopee, Blibli, dan Lazada, grouping pintar, dan insight AI dalam interface biru yang terasa hidup.</p>
          </div>

          <form onSubmit={onSubmit} className="visual-card" style={{ padding: 12, display: "grid", gap: 10, position: "relative", zIndex: 2 }}>
            <div style={{ display: "grid", gap: 12 }}>
              <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "end", flexWrap: "wrap" }}>
                <div>
                  <span className="text-faint" style={{ fontSize: 11, fontWeight: 900, letterSpacing: ".12em", textTransform: "uppercase" }}>1. Pilih marketplace</span>
                  <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--text-muted)" }}>Setiap marketplace punya tingkat kesiapan berbeda.</p>
                </div>
                <Link href="/pengaturan" className="ghost-button" style={{ padding: "8px 11px", fontSize: 12, textDecoration: "none" }}>Atur cookie</Link>
              </div>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(4, minmax(0, 1fr))", gap: 8 }} className="mobile-stack">
                {MARKETPLACES.map((item) => {
                  const active = marketplace === item.id;
                  return (
                    <button key={item.id} type="button" className={active ? "primary-button" : "ghost-button"} onClick={() => setMarketplace(item.id)} aria-pressed={active} style={{ padding: "12px 14px", justifyContent: "flex-start", textAlign: "left", minHeight: 82 }}>
                      <span style={{ display: "grid", gap: 4 }}>
                        <strong>{item.label}</strong>
                        <small style={{ opacity: .78, fontWeight: 750 }}>{item.helper}</small>
                        <small style={{ opacity: active ? .92 : .62, fontWeight: 700 }}>{item.readiness}</small>
                      </span>
                    </button>
                  );
                })}
              </div>
              <div className="visual-card" style={{ padding: 14, display: "grid", gap: 12, background: "linear-gradient(135deg, color-mix(in oklab, var(--accent) 12%, var(--surface)), var(--surface))" }}>
                <div style={{ display: "flex", justifyContent: "space-between", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
                  <div>
                    <span className={`marketplace-badge ${marketplaceClass(selectedMarketplace.id)}`}>{selectedMarketplace.label}</span>
                    <h3 style={{ margin: "8px 0 3px", fontSize: 18, letterSpacing: "-.03em" }}>{selectedMarketplace.readiness}</h3>
                    <p className="text-muted" style={{ margin: 0, fontSize: 13, lineHeight: 1.45 }}>{selectedMarketplace.tone}</p>
                  </div>
                  {selectedMarketplace.needsSettings ? <Link href="/pengaturan" className="primary-button" style={{ padding: "9px 13px", minHeight: "auto", textDecoration: "none" }}>Isi cookie</Link> : <span className="marketplace-badge mp-default">Tidak perlu setup</span>}
                </div>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 8 }} className="mobile-stack">
                  {selectedMarketplace.steps.map((step, index) => (
                    <div key={step} style={{ border: "1px solid var(--border)", borderRadius: 16, padding: 11, background: "color-mix(in oklab, var(--surface) 72%, transparent)" }}>
                      <span className="text-faint" style={{ fontSize: 11, fontWeight: 900 }}>STEP {index + 1}</span>
                      <p style={{ margin: "5px 0 0", fontSize: 13, lineHeight: 1.35, fontWeight: 760 }}>{step}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
            <div style={{ display: "grid", gap: 8 }}>
              <span className="text-faint" style={{ fontSize: 11, fontWeight: 900, letterSpacing: ".12em", textTransform: "uppercase" }}>2. Masukkan produk</span>
              <div style={{ display: "grid", gridTemplateColumns: "1fr auto", gap: 10 }} className="mobile-stack">
              <input
                aria-label="Nama produk"
                aria-describedby={error ? "search-error" : undefined}
                className="field"
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder="Cari iPhone 15, monitor 27 inch, sepatu running..."
                style={{ padding: "18px 18px", fontSize: 16, borderRadius: 20 }}
              />
              <button className="primary-button" disabled={isPending} aria-busy={isPending} style={{ padding: "0 22px", minHeight: 58 }}>
                {isPending ? "Scanning..." : `Cari di ${selectedMarketplace.label}`}
              </button>
              </div>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
              <button type="button" className="ghost-button" onClick={() => setShowFilters((value) => !value)} aria-expanded={showFilters} aria-controls="search-filters" style={{ padding: "9px 12px", fontSize: 12, fontWeight: 800 }}>
                {showFilters ? "Tutup filter" : "3. Filter pintar"}
              </button>
              <span className="text-faint" style={{ fontSize: 12 }}>{pricePreview} · {maxItems} produk · {SORT_OPTIONS.find((item) => item.value === sortBy)?.label}</span>
            </div>

            {showFilters ? (
              <div id="search-filters" className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 10 }}>
                <input aria-label="Harga minimum" className="field" value={minPrice} onChange={(event) => setMinPrice(event.target.value)} placeholder="Min price" style={{ padding: "11px 12px" }} />
                <input aria-label="Harga maksimum" className="field" value={maxPrice} onChange={(event) => setMaxPrice(event.target.value)} placeholder="Max price" style={{ padding: "11px 12px" }} />
                <select aria-label="Urutan hasil" className="field" value={sortBy} onChange={(event) => setSortBy(event.target.value)} style={{ padding: "11px 12px" }}>{SORT_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>
                <select aria-label="Jumlah produk" className="field" value={maxItems} onChange={(event) => setMaxItems(Number(event.target.value))} style={{ padding: "11px 12px" }}>{MAX_ITEMS_OPTIONS.map((option) => <option key={option} value={option}>{option} produk</option>)}</select>
              </div>
            ) : null}
            {error ? <p id="search-error" role="alert" style={{ color: "var(--error)", fontSize: 13, margin: 0 }}>{error}</p> : null}
          </form>
        </div>

        <aside className="hero-panel" style={{ padding: 20, background: "linear-gradient(145deg, var(--ink), color-mix(in oklab, var(--accent) 24%, var(--ink)))", color: "#f6f8ef", display: "grid", alignContent: "space-between", gap: 20 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <span style={{ fontSize: 13, opacity: .72 }}>Blue Market Pulse</span>
            <span className="metric-pill" style={{ color: "#11140f", background: "#dcefc7" }}>Creative AI</span>
          </div>
          <div className="visual-card" style={{ padding: 18, color: "var(--text)", animation: "floaty 5s ease-in-out infinite" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start", gap: 12 }}>
              <div>
                <p style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 6 }}>Signal terbaik</p>
                <h2 style={{ fontSize: 26, lineHeight: 1, fontWeight: 950, letterSpacing: "-.05em" }}>iPhone 15 128GB</h2>
              </div>
              <span className="marketplace-badge mp-tokopedia">Multi-marketplace</span>
            </div>
            <div style={{ marginTop: 24, display: "grid", gap: 10 }}>
              {["Official store", "Rating 4.9", "Harga lebih sehat"].map((item, index) => (
                <div key={item} style={{ display: "flex", alignItems: "center", justifyContent: "space-between", borderTop: index ? "1px solid var(--border)" : 0, paddingTop: index ? 10 : 0 }}>
                  <span style={{ fontSize: 13, color: "var(--text-muted)" }}>{item}</span>
                  <strong>{index === 0 ? "✓" : index === 1 ? "12rb ulasan" : "-8%"}</strong>
                </div>
              ))}
            </div>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <div className="visual-card" style={{ padding: 14, color: "var(--text)" }}><p className="text-faint" style={{ fontSize: 11 }}>Smart groups</p><strong style={{ fontSize: 26 }}>24</strong></div>
            <div className="visual-card" style={{ padding: 14, color: "var(--text)" }}><p className="text-faint" style={{ fontSize: 11 }}>Deal spread</p><strong style={{ fontSize: 26 }}>18%</strong></div>
          </div>
        </aside>
      </section>

      {recent.length > 0 ? (
        <section className="soft-card" style={{ borderRadius: 28, padding: 16, display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
          <span className="text-faint" style={{ fontSize: 12, fontWeight: 850, letterSpacing: ".1em", textTransform: "uppercase" }}>Recent</span>
          {recent.map((item) => (
            <button key={item} type="button" className="ghost-button" onClick={() => setKeyword(item)} style={{ padding: "8px 14px", borderRadius: "var(--radius-xl)", fontSize: 13 }}>{item}</button>
          ))}
        </section>
      ) : null}
    </div>
  );
}


function detailHref(id: string, keyword: string, marketplace: string) {
  const params = new URLSearchParams({ q: keyword, mp: marketplace });
  return `/pencarian/${id}?${params.toString()}`;
}


async function waitForRunReadable(id: string) {
  for (let attempt = 0; attempt < 30; attempt++) {
    const run = await api.getRun(id).catch(() => null);
    if (run?.status === "SUCCEEDED" || run?.status === "FAILED") return;
    await new Promise((resolve) => setTimeout(resolve, attempt < 3 ? 700 : 1200));
  }
}
