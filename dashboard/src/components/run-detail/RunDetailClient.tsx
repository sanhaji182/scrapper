"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { api, type AIStatus } from "@/lib/api";
import type { AISummaryResult, Product, ProductGroup, Run } from "@/lib/types";
import { formatDate, formatDuration, marketplaceLabel, runKeyword } from "@/lib/format";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { ProductsGrid } from "./ProductsGrid";
import { GroupsView } from "./GroupsView";
import { AIInsightsPanel } from "./AIInsightsPanel";
import { RunTechDetails } from "./RunTechDetails";

type Tab = "produk" | "kelompok" | "ai" | "teknis";

export function RunDetailClient({ initialRun }: { initialRun: Run & { result?: Product[] } }) {
  const [run, setRun] = useState(initialRun);
  const [products, setProducts] = useState<Product[]>(initialRun.result ?? []);
  const [groups, setGroups] = useState<ProductGroup[]>([]);
  const [aiSummary, setAiSummary] = useState<AISummaryResult | null>(null);
  const [aiStatus, setAiStatus] = useState<"idle" | "loading" | "done" | "error">("idle");
  const [aiReadiness, setAiReadiness] = useState<AIStatus | null>(null);
  const [aiError, setAIError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>("produk");

  useEffect(() => {
    if (run.status !== "RUNNING" && run.status !== "QUEUED") return;
    let cancelled = false;
    let timeout: ReturnType<typeof setTimeout> | null = null;

    const poll = async () => {
      const updated = await api.getRun(run.id).catch(() => null);
      if (cancelled) return;
      if (updated) {
        setRun(updated);
        if (updated.result?.length) setProducts(updated.result);
        if (updated.status !== "RUNNING" && updated.status !== "QUEUED") return;
      }
      timeout = setTimeout(poll, 1200);
    };

    timeout = setTimeout(poll, 500);
    return () => {
      cancelled = true;
      if (timeout) clearTimeout(timeout);
    };
  }, [run.id, run.status]);

  const loadAI = useCallback(async () => {
    setAiStatus("loading");
    setAIError(null);
    try {
      const status = aiReadiness ?? await api.getAIStatus().catch(() => null);
      if (status && !status.ready) {
        setAiReadiness(status);
        setAIError(status.message);
        setAiStatus("error");
        return;
      }
      let nextGroups: ProductGroup[] = [];
      try {
        nextGroups = await api.getNormalized(run.id);
      } catch {
        await api.triggerNormalize(run.id);
        nextGroups = await api.getNormalized(run.id);
      }
      setGroups(nextGroups);

      let nextSummary: AISummaryResult | null = null;
      try {
        nextSummary = await api.getAISummary(run.id);
      } catch {
        nextSummary = await api.triggerAISummary(run.id);
      }
      setAiSummary(nextSummary);
      setAiStatus("done");
    } catch (err) {
      setAIError(err instanceof Error ? err.message : "Model AI gagal memproses produk.");
      setAiStatus("error");
    }
  }, [aiReadiness, run.id]);

  useEffect(() => {
    let ignore = false;
    api.getAIStatus()
      .then((status) => {
        if (ignore) return;
        setAiReadiness(status);
        if (!status.ready) setAIError(status.message);
      })
      .catch((err) => {
        if (ignore) return;
        const message = err instanceof Error ? err.message : "Status AI tidak bisa dicek.";
        setAiReadiness({ ready: false, message });
        setAIError(message);
      });
    return () => { ignore = true; };
  }, []);

  useEffect(() => {
    if (run.status !== "SUCCEEDED" || aiStatus !== "idle") return;
    if (aiReadiness && !aiReadiness.ready) return;
    const timeout = window.setTimeout(() => {
      void loadAI();
    }, 0);
    return () => window.clearTimeout(timeout);
  }, [aiReadiness, aiStatus, loadAI, run.status]);

  const tabs = useMemo(() => [
    { id: "produk" as const, label: "Semua Produk", badge: products.length ? String(products.length) : undefined },
    { id: "kelompok" as const, label: "Kelompok Serupa", badge: groups.length ? String(groups.length) : undefined },
    { id: "ai" as const, label: "🤖 Rekomendasi AI", badge: aiReadiness?.ready === false ? "setup" : aiSummary?.recommended_items?.length ? String(aiSummary.recommended_items.length) : undefined },
    { id: "teknis" as const, label: "Detail Teknis" },
  ], [aiReadiness?.ready, aiSummary?.recommended_items?.length, groups.length, products.length]);

  const loadingProducts = run.status === "QUEUED" || run.status === "RUNNING";

  return (
    <div className="shell-container" style={{ display: "grid", gap: 20 }}>
      <header className="soft-card" style={{ borderRadius: 26, padding: 20 }}>
        <Link href="/riwayat" className="text-muted" style={{ fontSize: 13, textDecoration: "none" }}>← Kembali ke riwayat</Link>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 16, flexWrap: "wrap", marginTop: 14 }}>
          <div>
            <h1 style={{ fontSize: "clamp(24px, 4vw, 46px)", lineHeight: 1, fontWeight: 900, letterSpacing: "-0.06em" }}>Hasil: &quot;{runKeyword(run)}&quot;</h1>
            <div className="text-muted" style={{ display: "flex", gap: 11, flexWrap: "wrap", alignItems: "center", marginTop: 12, fontSize: 13 }}>
              <StatusBadge status={run.status} />
              <span>{marketplaceLabel(run.marketplace)}</span>
              <span>{products.length || run.item_count || 0} produk</span>
              <span>{formatDate(run.created_at)}</span>
              <span>{formatDuration(run.started_at, run.finished_at)}</span>
            </div>
          </div>
          <Link href="/" className="primary-button" style={{ padding: "10px 15px", alignSelf: "start" }}>+ Pencarian baru</Link>
        </div>
      </header>

      {run.status === "FAILED" ? (
        <section className="soft-card" style={{ borderRadius: 22, padding: 16, display: "flex", justifyContent: "space-between", gap: 14, alignItems: "center", flexWrap: "wrap", borderColor: "rgba(239, 68, 68, .26)" }}>
          <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
            <span style={{ width: 42, height: 42, borderRadius: 16, display: "grid", placeItems: "center", background: "rgba(239, 68, 68, .10)", fontSize: 22 }} aria-hidden="true">!</span>
            <div>
              <p style={{ margin: 0, fontWeight: 900 }}>{friendlyRunError(run.error_message).title}</p>
              <p className="text-muted" style={{ margin: "4px 0 0", fontSize: 13 }}>{friendlyRunError(run.error_message).body}</p>
            </div>
          </div>
          <Link href="/" className="primary-button" style={{ padding: "10px 14px" }}>Coba pencarian lain</Link>
        </section>
      ) : null}

      {loadingProducts ? (
        <div className="soft-card" style={{ borderRadius: 20, padding: 14, display: "flex", alignItems: "center", gap: 12 }}>
          <span className="status-badge status-running"><span className="dot" />Sedang mencari</span>
          <p className="text-muted" style={{ fontSize: 13 }}>Worker sedang mengambil produk. Halaman ini auto-refresh.</p>
        </div>
      ) : null}

      {run.status === "SUCCEEDED" && aiReadiness?.ready === false ? (
        <section className="soft-card" style={{ borderRadius: 22, padding: 16, display: "flex", justifyContent: "space-between", gap: 14, alignItems: "center", flexWrap: "wrap", borderColor: "rgba(245, 158, 11, .32)" }}>
          <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
            <span style={{ width: 42, height: 42, borderRadius: 16, display: "grid", placeItems: "center", background: "rgba(245, 158, 11, .12)", fontSize: 22 }} aria-hidden="true">⚙️</span>
            <div>
              <p style={{ margin: 0, fontWeight: 900 }}>AI belum aktif</p>
              <p className="text-muted" style={{ margin: "4px 0 0", fontSize: 13 }}>{aiReadiness.message || "Isi API key atau pilih Ollama lokal agar rekomendasi AI bisa berjalan."}</p>
            </div>
          </div>
          <Link href="/pengaturan" className="primary-button" style={{ padding: "10px 14px" }}>Atur AI</Link>
        </section>
      ) : null}

      <div role="tablist" aria-label="Tampilan hasil pencarian" style={{ display: "flex", gap: 8, overflowX: "auto", paddingBottom: 3 }}>
        {tabs.map((tab) => (
          <button key={tab.id} role="tab" aria-selected={activeTab === tab.id} className={`tab-button ${activeTab === tab.id ? "active" : ""}`} onClick={() => setActiveTab(tab.id)}>
            {tab.label}{tab.badge ? ` · ${tab.badge}` : ""}
          </button>
        ))}
      </div>

      {activeTab === "produk" ? <ProductsGrid products={products} loading={loadingProducts} /> : null}
      {activeTab === "kelompok" ? <GroupsView groups={groups} /> : null}
      {activeTab === "ai" ? <AIInsightsPanel summary={aiSummary} status={aiStatus} errorMessage={aiError} products={products} /> : null}
      {activeTab === "teknis" ? <RunTechDetails run={run} /> : null}
    </div>
  );
}

function friendlyRunError(error?: string | null) {
  const message = error ?? "";
  const lower = message.toLowerCase();
  if (lower.includes("shopee") && (lower.includes("403") || lower.includes("proxy"))) {
    return {
      title: "Shopee memblokir request otomatis",
      body: "Scraper sudah mencoba anonymous session, tetapi IP ini tetap ditolak Shopee. Isi proxy residential/mobile di PROXY_LIST lalu rebuild worker.",
    };
  }
  return {
    title: "Pencarian gagal",
    body: message || "Worker gagal menyelesaikan pencarian. Detail teknis tersedia di tab Detail Teknis.",
  };
}
