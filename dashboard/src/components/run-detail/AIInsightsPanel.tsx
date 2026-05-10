"use client";

import Link from "next/link";
import type { AISummaryResult, Product } from "@/lib/types";
import { formatRupiah, marketplaceClass, marketplaceLabel } from "@/lib/format";
import { MarketplaceExitButton } from "@/components/ui/MarketplaceExitButton";

const BADGES = ["Pilihan terbaik", "Termurah", "Paling aman", "Best value", "Top rating"];

export function AIInsightsPanel({
  summary,
  status,
  errorMessage,
  products,
}: {
  summary: AISummaryResult | null;
  status: "idle" | "loading" | "done" | "error";
  errorMessage?: string | null;
  products: Product[];
}) {
  if (status === "loading" || status === "idle") {
    return (
      <div style={{ padding: "32px 0", display: "grid", gap: 18 }}>
        <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
          <span style={{ fontSize: 30 }}>🤖</span>
          <div><p style={{ fontWeight: 850 }}>Menganalisis produk...</p><p className="text-muted" style={{ fontSize: 13 }}>AI sedang membaca {products.length} produk dan membandingkan value.</p></div>
        </div>
        <div className="product-grid">{Array.from({ length: 4 }).map((_, index) => <div key={index} className="skeleton-box" style={{ height: 156, borderRadius: 20 }} />)}</div>
      </div>
    );
  }

  if (status === "error") {
    const message = friendlyAIError(errorMessage);
    return (
      <div className="soft-card" style={{ borderRadius: 26, padding: 30, textAlign: "center", display: "grid", gap: 12, justifyItems: "center" }}>
        <div style={{ width: 58, height: 58, borderRadius: 22, display: "grid", placeItems: "center", background: "var(--accent-dim)", color: "var(--accent-strong)", fontSize: 28 }}>✦</div>
        <div>
          <p style={{ fontWeight: 900, fontSize: 20, margin: 0 }}>{message.title}</p>
          <p className="text-muted" style={{ fontSize: 13, margin: "8px auto 0", maxWidth: 520, lineHeight: 1.6 }}>{message.body}</p>
        </div>
        {errorMessage ? <code className="soft-panel" style={{ borderRadius: 14, padding: "8px 10px", fontSize: 11, color: "var(--text-muted)", maxWidth: "100%", overflowWrap: "anywhere" }}>{errorMessage}</code> : null}
        <Link href="/pengaturan" className="primary-button" style={{ padding: "11px 16px", marginTop: 4 }}>Buka Pengaturan AI</Link>
      </div>
    );
  }

  if (!summary) return null;
  const productMap = new Map(products.map((product) => [product.id, product]));

  return (
    <div style={{ display: "grid", gap: 18 }}>
      <section className="soft-card" style={{ borderRadius: 22, padding: 20, borderLeft: "4px solid var(--accent)" }}>
        <p className="accent-text" style={{ fontSize: 12, fontWeight: 850, letterSpacing: "0.1em", textTransform: "uppercase", marginBottom: 8 }}>🤖 Ringkasan AI</p>
        <p style={{ lineHeight: 1.75 }}>{summary.summary_text}</p>
      </section>

      {summary.recommended_items?.length ? (
        <section style={{ display: "grid", gap: 12 }}>
          <p className="text-faint" style={{ fontSize: 12, fontWeight: 850, letterSpacing: "0.1em", textTransform: "uppercase" }}>Rekomendasi teratas</p>
          <div className="product-grid">
            {summary.recommended_items.map((item, index) => {
              const product = productMap.get(item.product_id);
              return (
                <article key={`${item.group_id}-${item.product_id}-${index}`} className="soft-card" style={{ borderRadius: 22, padding: 16, display: "grid", gap: 11 }}>
                  <div style={{ display: "flex", justifyContent: "space-between", gap: 10 }}>
                    <span className="marketplace-badge mp-tokopedia">{BADGES[index % BADGES.length]}</span>
                    {product ? <span className={`marketplace-badge ${marketplaceClass(product.marketplace)}`}>{marketplaceLabel(product.marketplace)}</span> : null}
                  </div>
                  <h3 style={{ fontSize: 15, fontWeight: 850, lineHeight: 1.35 }}>{product?.name ?? item.product_id}</h3>
                  {product ? <p className="accent-text" style={{ fontSize: 18, fontWeight: 900 }}>{formatRupiah(product.price)}</p> : null}
                  <p className="text-muted" style={{ fontSize: 13, lineHeight: 1.6 }}>{item.reason}</p>
                  {product ? <MarketplaceExitButton url={product.url} label="Buka produk →" productName={product.name} shopName={product.shop_name} style={{ padding: "8px 10px", textAlign: "center", fontSize: 12, fontWeight: 800 }} /> : null}
                </article>
              );
            })}
          </div>
        </section>
      ) : null}
    </div>
  );
}

function friendlyAIError(error?: string | null) {
  const value = (error ?? "").toLowerCase();
  if (value.includes("api key") || value.includes("401") || value.includes("unauthorized")) {
    return {
      title: "AI belum aktif",
      body: "API key belum diisi atau tidak valid. Isi API key di Pengaturan AI, lalu klik Test koneksi.",
    };
  }
  if (value.includes("model") || value.includes("404")) {
    return {
      title: "Model AI bermasalah",
      body: "Model yang dipilih kemungkinan tidak tersedia untuk provider ini. Ganti model di Pengaturan AI lalu test lagi.",
    };
  }
  if (value.includes("timeout") || value.includes("deadline")) {
    return {
      title: "AI terlalu lama merespons",
      body: "Provider sedang lambat atau timeout terlalu pendek. Coba naikkan timeout di Pengaturan AI.",
    };
  }
  return {
    title: "Analisis AI belum tersedia",
    body: "Provider AI belum siap atau gagal memproses produk. Kamu tetap bisa memakai tab produk dan kelompok.",
  };
}
