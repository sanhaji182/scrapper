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
    icon: "T",
    helper: "Paling stabil",
    readiness: "Langsung cari",
    tone: "Search publik stabil tanpa setup tambahan.",
    steps: ["Input keyword", "Pilih jumlah", "Jalankan scraping"],
    needsSettings: false,
  },
  {
    id: "shopee",
    label: "Shopee",
    icon: "S",
    helper: "Cookie disarankan",
    readiness: "Butuh sesi jika 403",
    tone: "Cookie browser membantu request terlihat seperti sesi valid.",
    steps: ["Buka Shopee", "Copy Cookie", "Paste di Settings"],
    needsSettings: true,
  },
  {
    id: "blibli",
    label: "Blibli",
    icon: "B",
    helper: "Biasanya langsung",
    readiness: "Cookie opsional",
    tone: "Mulai tanpa cookie. Proxy membantu kalau 403 berulang.",
    steps: ["Coba langsung", "30–100 produk", "Proxy jika perlu"],
    needsSettings: false,
  },
  {
    id: "lazada",
    label: "Lazada",
    icon: "L",
    helper: "Paling ketat",
    readiness: "Session + proxy ideal",
    tone: "Sering captcha; cookie dan residential proxy paling stabil.",
    steps: ["Selesaikan captcha", "Copy Cookie", "Gunakan proxy"],
    needsSettings: true,
  },
];

const emptyRecentSearches: string[] = [];
let recentSnapshotRaw = "[]";
let recentSnapshotValue: string[] = emptyRecentSearches;

type MarketplaceItem = (typeof MARKETPLACES)[number];

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
    if (!min && !max) return "Semua harga";
    if (min && max) return `${formatRupiahShort(min)} – ${formatRupiahShort(max)}`;
    if (min) return `Mulai ${formatRupiahShort(min)}`;
    return `Hingga ${formatRupiahShort(max)}`;
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
    <div className="ps-dashboard shell-container">
      <main className="ps-main-column">
        <section className="ps-hero-card">
          <div className="ps-hero-topline">
            <span className="hero-eyebrow">AI Marketplace Intelligence</span>
            <Link href="/pengaturan" className="ps-mini-link">Cookie & AI Settings</Link>
          </div>
          <h1 className="ps-hero-title">Cari produk, <span>bandingkan harga</span>, ambil keputusan lebih cepat.</h1>
          <p className="ps-hero-subtitle">Scrape Tokopedia, Shopee, Blibli, dan Lazada dalam workflow yang terasa sederhana—dengan AI normalizer dan insight live untuk mempercepat riset harga.</p>
          <div className="ps-stat-row">
            <StatCard icon="◈" value="4" label="Marketplace" />
            <StatCard icon="⌁" value="100" label="Produk / Job" />
            <StatCard icon="✦" value="AI" label="Normalizer" />
          </div>
        </section>

        <form onSubmit={onSubmit} className="ps-search-console">
          <SectionHeader kicker="Source Selection" title="Pilih marketplace" copy="Pilih sumber data. Kami tampilkan kebutuhan setup supaya flow scraping terasa jelas." />
          <div className="ps-marketplace-grid">
            {MARKETPLACES.map((item) => <MarketplaceCard key={item.id} item={item} active={marketplace === item.id} onSelect={() => setMarketplace(item.id)} />)}
          </div>

          <div className="ps-workflow-strip" aria-label="Workflow pencarian">
            {selectedMarketplace.steps.map((step, index) => <WorkflowStep key={step} index={index + 1} label={step} />)}
          </div>

          <div className="ps-search-area">
            <div className="ps-search-bar-wrap">
              <span className="ps-search-icon">⌕</span>
              <input aria-label="Nama produk" aria-describedby={error ? "search-error" : undefined} className="ps-search-input" value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="Cari iPhone 15, RTX 5060, Kindle 11..." />
            </div>
            <button className="ps-cta-button" disabled={isPending} aria-busy={isPending}>{isPending ? "Scanning..." : `Cari di ${selectedMarketplace.label}`}</button>
          </div>

          <div className="ps-filter-row">
            <button type="button" className="ps-filter-button" onClick={() => setShowFilters((value) => !value)} aria-expanded={showFilters} aria-controls="search-filters">Smart filter</button>
            <span>{pricePreview} · {maxItems} produk · {SORT_OPTIONS.find((item) => item.value === sortBy)?.label}</span>
          </div>

          {showFilters ? (
            <div id="search-filters" className="ps-filter-grid">
              <input aria-label="Harga minimum" className="field" value={minPrice} onChange={(event) => setMinPrice(event.target.value)} placeholder="Min price" />
              <input aria-label="Harga maksimum" className="field" value={maxPrice} onChange={(event) => setMaxPrice(event.target.value)} placeholder="Max price" />
              <select aria-label="Urutan hasil" className="field" value={sortBy} onChange={(event) => setSortBy(event.target.value)}>{SORT_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>
              <select aria-label="Jumlah produk" className="field" value={maxItems} onChange={(event) => setMaxItems(Number(event.target.value))}>{MAX_ITEMS_OPTIONS.map((option) => <option key={option} value={option}>{option} produk</option>)}</select>
            </div>
          ) : null}
          {error ? <p id="search-error" role="alert" className="ps-error-message">{error}</p> : null}
        </form>

        <section className="ps-lower-grid">
          <div className="ps-playbook-card">
            <SectionHeader kicker="Marketplace Playbook" title="Tips sebelum scraping" copy="Ringkasan setup tiap marketplace agar user tahu apa yang diperlukan." />
            <div className="ps-playbook-grid">
              {MARKETPLACES.map((item) => <PlaybookCard key={item.id} item={item} />)}
            </div>
          </div>
          <AIInsightCard />
        </section>

        {recent.length > 0 ? (
          <section className="ps-recent-card">
            <span>Recent searches</span>
            <div>{recent.map((item) => <button key={item} type="button" onClick={() => setKeyword(item)}>{item}</button>)}</div>
          </section>
        ) : null}
      </main>

      <aside className="ps-right-panel">
        <AILiveDecisionBoard selected={selectedMarketplace} keyword={keyword} maxItems={maxItems} pricePreview={pricePreview} onRun={() => document.querySelector<HTMLButtonElement>(".ps-cta-button")?.click()} />
      </aside>
    </div>
  );
}

function SectionHeader({ kicker, title, copy }: { kicker: string; title: string; copy: string }) {
  return <div className="ps-section-header"><span>{kicker}</span><h2>{title}</h2><p>{copy}</p></div>;
}

function StatCard({ icon, value, label }: { icon: string; value: string; label: string }) {
  return <div className="ps-stat-card"><span>{icon}</span><strong>{value}</strong><small>{label}</small></div>;
}

function MarketplaceCard({ item, active, onSelect }: { item: MarketplaceItem; active: boolean; onSelect: () => void }) {
  return (
    <button type="button" onClick={onSelect} aria-pressed={active} className={`ps-market-card ${active ? "active" : ""}`}>
      <div className="ps-market-icon">{item.icon}</div>
      <div><strong>{item.label}</strong><p>{item.helper}</p></div>
      <span className={`marketplace-badge ${marketplaceClass(item.id)}`}>{item.readiness}</span>
    </button>
  );
}

function WorkflowStep({ index, label }: { index: number; label: string }) {
  return <div className="ps-workflow-step"><span>STEP {index}</span><strong>{label}</strong></div>;
}

function PlaybookCard({ item }: { item: MarketplaceItem }) {
  return <div className="ps-playbook-item"><span className={`marketplace-badge ${marketplaceClass(item.id)}`}>{item.label}</span><strong>{item.readiness}</strong><p>{item.needsSettings ? "Siapkan cookie browser di Settings untuk stabilitas lebih baik." : "Bisa langsung dicoba tanpa konfigurasi tambahan."}</p></div>;
}

function AIInsightCard() {
  return (
    <div className="ps-ai-insight-card">
      <div className="ps-ai-orb">✦</div>
      <span>AI Insight Layer</span>
      <h3>Assisted price intelligence</h3>
      <ul>
        <li>Normalisasi harga</li>
        <li>Deteksi outlier</li>
        <li>Rekomendasi cepat</li>
      </ul>
    </div>
  );
}

function AILiveDecisionBoard({ selected, keyword, maxItems, pricePreview, onRun }: { selected: MarketplaceItem; keyword: string; maxItems: number; pricePreview: string; onRun: () => void }) {
  return (
    <div className="ps-ai-board">
      <div className="ps-ai-board-head"><span>AI Live Decision Board</span><strong>Live</strong></div>
      <div className="ps-preview-product">
        <span className="ps-preview-glow" />
        <small>Preview produk</small>
        <h2>{keyword.trim() || "iPhone 15"}</h2>
        <p>AI akan membantu membaca pola harga, grouping produk serupa, dan memberi sinyal keputusan.</p>
      </div>
      <div className="ps-board-metrics">
        <MetricRow label="Source status" value={selected.readiness} />
        <MetricRow label="Scope pencarian" value={`${maxItems} produk`} />
        <MetricRow label="Filter harga" value={pricePreview} />
      </div>
      <button type="button" onClick={onRun} className="ps-glow-button">Jalankan Pencarian</button>
      <div className="ps-vertical-flow">
        {[["01", "Search", "Submit job"], ["02", "Worker", "Scrape data"], ["03", "AI", "Normalize & insight"]].map(([num, title, copy]) => <div key={title}><span>{num}</span><strong>{title}</strong><small>{copy}</small></div>)}
      </div>
    </div>
  );
}

function MetricRow({ label, value }: { label: string; value: string }) {
  return <div><span>{label}</span><strong>{value}</strong></div>;
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
