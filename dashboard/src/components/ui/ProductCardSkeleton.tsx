export function ProductCardSkeleton() {
  return (
    <div className="soft-card" style={{ borderRadius: "var(--radius-lg)", overflow: "hidden" }}>
      <div className="skeleton-box" style={{ aspectRatio: "1.15" }} />
      <div style={{ padding: 12, display: "flex", flexDirection: "column", gap: 8 }}>
        <div className="skeleton-box" style={{ height: 14, width: "82%" }} />
        <div className="skeleton-box" style={{ height: 12, width: "54%" }} />
        <div className="skeleton-box" style={{ height: 18, width: "66%", marginTop: 4 }} />
        <div className="skeleton-box" style={{ height: 34, marginTop: 8 }} />
      </div>
    </div>
  );
}
