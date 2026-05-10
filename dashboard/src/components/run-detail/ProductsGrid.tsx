"use client";

import { useMemo, useState } from "react";
import type { Product } from "@/lib/types";
import { formatCount, formatRupiah, marketplaceClass, marketplaceLabel } from "@/lib/format";
import { ProductCardSkeleton } from "@/components/ui/ProductCardSkeleton";
import { MarketplaceExitButton } from "@/components/ui/MarketplaceExitButton";

function productScore(product: Product) {
  return (product.rating || 0) * 100 + (product.count_review || 0) / 20 + (product.is_official_store ? 75 : 0);
}

export function ProductsGrid({ products, loading = false }: { products: Product[]; loading?: boolean }) {
  const [query, setQuery] = useState("");
  const [officialOnly, setOfficialOnly] = useState(false);
  const [sort, setSort] = useState("value");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const next = products.filter((product) => {
      if (officialOnly && !product.is_official_store) return false;
      if (!q) return true;
      return `${product.name} ${product.shop_name} ${product.shop_city}`.toLowerCase().includes(q);
    });
    next.sort((a, b) => {
      if (sort === "price_asc") return a.price - b.price;
      if (sort === "price_desc") return b.price - a.price;
      if (sort === "rating") return (b.rating || 0) - (a.rating || 0);
      return productScore(b) - productScore(a);
    });
    return next;
  }, [officialOnly, products, query, sort]);

  if (loading) {
    return <div className="product-grid">{Array.from({ length: 8 }).map((_, index) => <ProductCardSkeleton key={index} />)}</div>;
  }

  return (
    <div style={{ display: "grid", gap: 14 }}>
      <div className="soft-card mobile-stack" style={{ borderRadius: 20, padding: 12, display: "grid", gridTemplateColumns: "1fr auto auto", gap: 10 }}>
        <input aria-label="Filter produk" className="field" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filter nama produk, toko, atau kota" style={{ padding: "10px 12px" }} />
        <select aria-label="Urutkan produk" className="field" value={sort} onChange={(event) => setSort(event.target.value)} style={{ padding: "10px 12px" }}>
          <option value="value">Best value</option>
          <option value="price_asc">Harga termurah</option>
          <option value="price_desc">Harga tertinggi</option>
          <option value="rating">Rating tertinggi</option>
        </select>
        <button className={`ghost-button ${officialOnly ? "active" : ""}`} aria-pressed={officialOnly} onClick={() => setOfficialOnly((value) => !value)} style={{ padding: "10px 12px" }}>
          {officialOnly ? "✓ Toko resmi" : "Toko resmi"}
        </button>
      </div>

      {filtered.length === 0 ? (
        <div className="soft-card" style={{ borderRadius: 22, padding: 30, textAlign: "center" }}>
          <p className="text-muted">Tidak ada produk yang cocok dengan filter.</p>
        </div>
      ) : (
        <div className="product-grid">
          {filtered.map((product, index) => <ProductCard key={`${product.marketplace}-${product.id}-${index}`} product={product} />)}
        </div>
      )}
    </div>
  );
}

function ProductCard({ product }: { product: Product }) {
  const [imageFailed, setImageFailed] = useState(false);

  return (
    <article className="soft-card product-card">
      <div style={{ aspectRatio: "1.12", background: "var(--surface-2)", position: "relative", overflow: "hidden" }}>
        {product.image_url && !imageFailed ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={product.image_url} alt={product.name} onError={() => setImageFailed(true)} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
        ) : (
          <div
            style={{
              display: "grid",
              placeItems: "center",
              height: "100%",
              padding: 18,
              color: "var(--accent-strong)",
              background: "radial-gradient(circle at 30% 20%, var(--accent-dim), transparent 12rem), var(--surface-2)",
              textAlign: "center",
            }}
          >
            <div>
              <div style={{ fontSize: 42, lineHeight: 1 }}>⌁</div>
              <p className="text-muted" style={{ margin: "8px 0 0", fontSize: 12, lineHeight: 1.35 }}>Gambar marketplace tidak tersedia</p>
            </div>
          </div>
        )}
        <div style={{ position: "absolute", left: 10, top: 10 }} className={`marketplace-badge ${marketplaceClass(product.marketplace)}`}>{marketplaceLabel(product.marketplace)}</div>
        {product.discount_percent > 0 ? <div style={{ position: "absolute", right: 10, top: 10, background: "rgba(167,95,90,.92)", color: "white", borderRadius: 99, padding: "3px 8px", fontSize: 11, fontWeight: 800 }}>-{product.discount_percent}%</div> : null}
      </div>
      <div style={{ padding: 13, display: "grid", gap: 9 }}>
        <h3 title={product.name} style={{ fontSize: 13.5, lineHeight: 1.35, fontWeight: 750, minHeight: 38, display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>{product.name}</h3>
        <div>
          <p style={{ fontSize: 17, fontWeight: 850, color: "var(--accent-strong)" }}>{formatRupiah(product.price)}</p>
          {product.original_price > product.price ? <p className="text-faint" style={{ fontSize: 11, textDecoration: "line-through" }}>{formatRupiah(product.original_price)}</p> : null}
        </div>
        <div className="text-muted" style={{ display: "flex", flexWrap: "wrap", gap: 7, fontSize: 11.5 }}>
          {product.rating > 0 ? <span>★ {product.rating.toFixed(1)}</span> : null}
          {product.count_review > 0 ? <span>{formatCount(product.count_review)} ulasan</span> : null}
          {product.sold > 0 ? <span>{formatCount(product.sold)} terjual</span> : null}
        </div>
        <div className="text-muted" style={{ fontSize: 11.5, display: "grid", gap: 2 }}>
          <span>{product.shop_name || "Toko"}{product.is_official_store ? " • Resmi" : ""}</span>
          <span>{product.shop_city || "Indonesia"}</span>
        </div>
        <MarketplaceExitButton url={product.url} productName={product.name} shopName={product.shop_name} style={{ padding: "8px 10px", textAlign: "center", fontSize: 12, fontWeight: 750 }} />
      </div>
    </article>
  );
}
