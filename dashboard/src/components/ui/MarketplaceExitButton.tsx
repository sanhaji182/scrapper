"use client";

import { useId, useState } from "react";
import { createPortal } from "react-dom";

interface MarketplaceExitButtonProps {
  url: string;
  label?: string;
  productName?: string;
  shopName?: string;
  className?: string;
  style?: React.CSSProperties;
}

export function MarketplaceExitButton({
  url,
  label = "Buka marketplace →",
  productName,
  shopName,
  className = "ghost-button",
  style,
}: MarketplaceExitButtonProps) {
  const [open, setOpen] = useState(false);
  const titleID = useId();
  const host = safeHost(url);

  return (
    <>
      <button type="button" className={className} style={style} onClick={() => setOpen(true)}>
        {label}
      </button>

      {open && typeof document !== "undefined" ? createPortal(

        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleID}
          onClick={() => setOpen(false)}
          style={{
            position: "fixed",
            inset: 0,
            zIndex: 80,
            display: "grid",
            placeItems: "center",
            padding: 18,
            background: "rgba(3, 10, 24, 0.62)",
            backdropFilter: "blur(16px)",
          }}
        >
          <div
            className="soft-card"
            onClick={(event) => event.stopPropagation()}
            style={{
              width: "min(480px, 100%)",
              borderRadius: 28,
              overflow: "hidden",
              boxShadow: "var(--shadow-soft)",
            }}
          >
            <div
              style={{
                padding: 20,
                background: "linear-gradient(135deg, #061b44, #115fd6 58%, #18b7d8)",
                color: "white",
              }}
            >
              <div style={{ display: "flex", justifyContent: "space-between", gap: 14, alignItems: "flex-start" }}>
                <div>
                  <p style={{ margin: 0, fontSize: 11, fontWeight: 900, letterSpacing: ".14em", textTransform: "uppercase", opacity: 0.78 }}>
                    Marketplace
                  </p>
                  <h2 id={titleID} style={{ margin: "9px 0 0", fontSize: 28, lineHeight: 1, fontWeight: 950, letterSpacing: "-.05em" }}>
                    Buka situs asli?
                  </h2>
                </div>
                <button
                  type="button"
                  aria-label="Tutup modal"
                  onClick={() => setOpen(false)}
                  style={{
                    width: 38,
                    height: 38,
                    border: "1px solid rgba(255,255,255,.28)",
                    borderRadius: 14,
                    background: "rgba(255,255,255,.12)",
                    color: "white",
                    fontSize: 20,
                    cursor: "pointer",
                    flex: "0 0 auto",
                  }}
                >
                  ×
                </button>
              </div>
            </div>

            <div style={{ padding: 18, display: "grid", gap: 14 }}>
              <div className="soft-panel" style={{ borderRadius: 20, padding: 14, display: "grid", gap: 6 }}>
                <p style={{ margin: 0, fontSize: 15, fontWeight: 880, lineHeight: 1.35, display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
                  {productName || "Produk marketplace"}
                </p>
                {shopName ? <p className="text-muted" style={{ margin: 0, fontSize: 13 }}>{shopName}</p> : null}
                <p className="text-faint" style={{ margin: 0, fontSize: 12, wordBreak: "break-word" }}>{host}</p>
              </div>

              <p className="text-muted" style={{ margin: 0, fontSize: 13.5, lineHeight: 1.6 }}>
                Kamu akan membuka marketplace di tab baru. Cek ulang harga, stok, promo, dan ongkir sebelum transaksi.
              </p>

              <div className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "1fr 1.25fr", gap: 10 }}>
                <button type="button" className="ghost-button" onClick={() => setOpen(false)} style={{ padding: "12px 14px", fontWeight: 850, width: "100%" }}>
                  Batal
                </button>
                <a
                  href={url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="primary-button"
                  onClick={() => setOpen(false)}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    padding: "12px 14px",
                    textAlign: "center",
                    width: "100%",
                  }}
                >
                  Buka marketplace ↗
                </a>
              </div>
            </div>
          </div>
        </div>
        , document.body) : null}
    </>
  );
}

function safeHost(url: string) {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}
