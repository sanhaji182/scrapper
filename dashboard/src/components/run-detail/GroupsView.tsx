"use client";

import { useState } from "react";
import type { ProductGroup } from "@/lib/types";
import { formatCount, formatRupiah, formatRupiahShort, marketplaceClass, marketplaceLabel } from "@/lib/format";
import { MarketplaceExitButton } from "@/components/ui/MarketplaceExitButton";

export function GroupsView({ groups }: { groups: ProductGroup[] }) {
  if (groups.length === 0) {
    return <div className="soft-card" style={{ borderRadius: 22, padding: 30, textAlign: "center" }}><p className="text-muted">Belum ada kelompok produk. Normalisasi akan berjalan setelah hasil tersedia.</p></div>;
  }
  return <div style={{ display: "grid", gap: 14 }}>{groups.map((group) => <GroupCard key={group.group_id} group={group} />)}</div>;
}

function GroupCard({ group }: { group: ProductGroup }) {
  const [expanded, setExpanded] = useState(false);
  const range = Math.max(1, group.max_price - group.min_price);
  const bestItem = group.items.find((item) => item.product_id === group.best_price_id) ?? group.items[0];

  return (
    <article className="soft-card" style={{ borderRadius: 22, overflow: "hidden" }}>
      <div style={{ padding: 18 }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 16, flexWrap: "wrap" }}>
          <div style={{ flex: 1, minWidth: 220 }}>
            <h3 style={{ fontSize: 17, fontWeight: 850 }}>{group.canonical_name || group.group_id}</h3>
            <p className="text-muted" style={{ fontSize: 12, marginTop: 4 }}>{group.items.length} penawaran • {group.category_path || "Kategori belum pasti"}</p>
            {group.important_specs?.length ? <div style={{ display: "flex", flexWrap: "wrap", gap: 6, marginTop: 10 }}>{group.important_specs.slice(0, 5).map((spec) => <span key={spec} className="ghost-button" style={{ padding: "3px 8px", fontSize: 11 }}>{spec}</span>)}</div> : null}
          </div>
          <div style={{ textAlign: "right" }}>
            <p style={{ fontSize: 20, fontWeight: 900 }}>{formatRupiahShort(group.min_price)}</p>
            <p className="text-muted" style={{ fontSize: 12 }}>– {formatRupiahShort(group.max_price)}</p>
            <p className="text-faint" style={{ fontSize: 11 }}>rata-rata {formatRupiahShort(group.avg_price)}</p>
          </div>
        </div>

        <div style={{ margin: "16px 0 12px" }}>
          <div style={{ height: 5, background: "var(--surface-3)", borderRadius: 99, position: "relative" }}>
            {group.items.map((item, index) => {
              const pct = ((item.price - group.min_price) / range) * 92 + 2;
              return <span key={`${item.product_id}-${index}`} title={`${item.shop_name}: ${formatRupiah(item.price)}`} style={{ position: "absolute", left: `${pct}%`, top: "50%", width: 11, height: 11, borderRadius: "50%", transform: "translate(-50%, -50%)", background: item.product_id === group.best_price_id ? "var(--accent)" : "var(--text-faint)", boxShadow: item.product_id === group.best_price_id ? "0 0 0 5px var(--accent-dim)" : "none" }} />;
            })}
          </div>
          <div className="text-faint" style={{ display: "flex", justifyContent: "space-between", marginTop: 6, fontSize: 11 }}><span>Termurah</span><span>Termahal</span></div>
        </div>

        {bestItem ? (
          <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap", border: "1px solid rgba(111,141,118,.22)", background: "var(--accent-dim)", borderRadius: 16, padding: "10px 12px" }}>
            <span className="accent-text" style={{ fontSize: 11, fontWeight: 850, letterSpacing: "0.08em", textTransform: "uppercase" }}>Harga terbaik</span>
            <span className={`marketplace-badge ${marketplaceClass(bestItem.marketplace)}`}>{marketplaceLabel(bestItem.marketplace)}</span>
            <strong>{formatRupiah(bestItem.price)}</strong>
            <span className="text-muted" style={{ fontSize: 12 }}>{bestItem.shop_name}</span>
            {bestItem.is_official_store ? <span className="marketplace-badge mp-tokopedia">✓ Resmi</span> : null}
            <MarketplaceExitButton url={bestItem.url} label="Lihat →" productName={bestItem.name} shopName={bestItem.shop_name} className="accent-text" style={{ marginLeft: "auto", fontSize: 12, fontWeight: 800, border: 0, background: "transparent", padding: 0 }} />
          </div>
        ) : null}
      </div>

      <button onClick={() => setExpanded((value) => !value)} style={{ width: "100%", padding: "11px 18px", border: 0, borderTop: "1px solid var(--border)", background: "transparent", color: "var(--text-muted)", cursor: "pointer", fontWeight: 750, fontSize: 12 }}>
        {expanded ? "Sembunyikan penawaran" : `Lihat semua ${group.items.length} penawaran`} {expanded ? "↑" : "↓"}
      </button>

      {expanded ? (
        <div style={{ borderTop: "1px solid var(--border)" }}>
          {group.items.map((item, index) => (
            <div key={`${item.product_id}-${index}`} style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap", padding: "12px 18px", borderBottom: index < group.items.length - 1 ? "1px solid var(--border)" : "none", background: item.product_id === group.best_price_id ? "var(--accent-dim)" : "transparent" }}>
              <span className={`marketplace-badge ${marketplaceClass(item.marketplace)}`}>{marketplaceLabel(item.marketplace)}</span>
              <div style={{ flex: 1, minWidth: 150 }}><p style={{ fontWeight: 750, fontSize: 13 }}>{item.shop_name}</p><p className="text-muted" style={{ fontSize: 11 }}>{item.shop_city} • ★ {item.rating?.toFixed?.(1) ?? "-"} • {formatCount(item.count_review)} ulasan</p></div>
              <strong>{formatRupiah(item.price)}</strong>
              <MarketplaceExitButton url={item.url} label="Buka" productName={item.name} shopName={item.shop_name} style={{ padding: "6px 10px", fontSize: 12 }} />
            </div>
          ))}
        </div>
      ) : null}
    </article>
  );
}
