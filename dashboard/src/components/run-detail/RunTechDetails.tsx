import type { Run } from "@/lib/types";
import { formatDate, formatDuration, marketplaceLabel, runKeyword } from "@/lib/format";
import { StatusBadge } from "@/components/ui/StatusBadge";

export function RunTechDetails({ run }: { run: Run }) {
  const rows = [
    ["Pencarian", runKeyword(run)],
    ["Marketplace", marketplaceLabel(run.marketplace)],
    ["Status", <StatusBadge key="status" status={run.status} />],
    ["Jumlah item", `${run.item_count || 0} produk`],
    ["Dibuat", formatDate(run.created_at)],
    ["Mulai", formatDate(run.started_at)],
    ["Selesai", formatDate(run.finished_at)],
    ["Durasi", formatDuration(run.started_at, run.finished_at)],
    ["Error", run.error_message || "-"],
  ];

  return (
    <div className="mobile-stack" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
      <section className="soft-card" style={{ borderRadius: 22, padding: 18 }}>
        <h2 style={{ fontSize: 17, fontWeight: 850, marginBottom: 12 }}>Metadata pencarian</h2>
        <div style={{ display: "grid", gap: 10 }}>
          {rows.map(([label, value]) => (
            <div key={String(label)} style={{ display: "flex", justifyContent: "space-between", gap: 12, borderBottom: "1px solid var(--border)", paddingBottom: 9 }}>
              <span className="text-muted" style={{ fontSize: 12 }}>{label}</span>
              <span style={{ fontSize: 12, textAlign: "right", fontWeight: 700 }}>{value}</span>
            </div>
          ))}
        </div>
      </section>
      <section className="soft-card" style={{ borderRadius: 22, padding: 18 }}>
        <h2 style={{ fontSize: 17, fontWeight: 850, marginBottom: 12 }}>Input teknis</h2>
        <pre style={{ margin: 0, whiteSpace: "pre-wrap", wordBreak: "break-word", color: "var(--text-muted)", fontSize: 12, lineHeight: 1.65 }}>{JSON.stringify(run.input ?? {}, null, 2)}</pre>
      </section>
    </div>
  );
}
