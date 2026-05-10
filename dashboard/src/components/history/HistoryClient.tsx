"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Run } from "@/lib/types";
import { formatDate, formatDuration, marketplaceLabel, runKeyword } from "@/lib/format";
import { StatusBadge } from "@/components/ui/StatusBadge";

export function HistoryClient({ initialRuns, total }: { initialRuns: Run[]; total: number }) {
  const [runs, setRuns] = useState(initialRuns);

  useEffect(() => {
    const interval = setInterval(async () => {
      const next = await api.getRuns(50, 0).catch(() => null);
      if (next) setRuns(next.runs);
    }, runs.some((run) => run.status === "QUEUED" || run.status === "RUNNING") ? 3000 : 15000);
    return () => clearInterval(interval);
  }, [runs]);

  return (
    <div className="shell-container" style={{ display: "grid", gap: 18 }}>
      <header style={{ display: "flex", alignItems: "end", justifyContent: "space-between", gap: 14, flexWrap: "wrap" }}>
        <div>
          <p className="accent-text" style={{ fontSize: 12, fontWeight: 800, letterSpacing: "0.12em", textTransform: "uppercase" }}>Riwayat pencarian</p>
          <h1 style={{ fontSize: "clamp(28px, 4vw, 48px)", letterSpacing: "-0.06em", fontWeight: 850 }}>Semua pencarian</h1>
          <p className="text-muted" style={{ marginTop: 4 }}>{total} pencarian tersimpan dari worker.</p>
        </div>
        <Link href="/" className="primary-button" style={{ padding: "10px 16px" }}>+ Cari produk</Link>
      </header>

      {runs.length === 0 ? (
        <div className="soft-card" style={{ borderRadius: 24, padding: 34, textAlign: "center" }}>
          <h2 style={{ fontSize: 22, fontWeight: 800 }}>Belum ada pencarian</h2>
          <p className="text-muted" style={{ marginTop: 8 }}>Mulai dari search hub untuk membuat job pertama.</p>
        </div>
      ) : (
        <div style={{ display: "grid", gap: 12 }}>
          {runs.map((run) => (
            <Link key={run.id} href={detailHref(run.id, runKeyword(run), run.marketplace)} className="soft-card" style={{ borderRadius: 20, padding: 16, textDecoration: "none", display: "block" }}>
              <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 14, flexWrap: "wrap" }}>
                <div style={{ minWidth: 220, flex: 1 }}>
                  <h2 style={{ fontSize: 17, fontWeight: 800, marginBottom: 8 }}>&quot;{runKeyword(run)}&quot;</h2>
                  <div style={{ display: "flex", gap: 10, flexWrap: "wrap", alignItems: "center", fontSize: 12 }} className="text-muted">
                    <StatusBadge status={run.status} />
                    <span>{marketplaceLabel(run.marketplace)}</span>
                    {run.item_count > 0 ? <span>{run.item_count} produk</span> : null}
                    <span>{formatDate(run.created_at)}</span>
                    <span>{formatDuration(run.started_at, run.finished_at)}</span>
                  </div>
                </div>
                <span className="ghost-button" style={{ padding: "8px 13px", fontSize: 12 }}>Lihat hasil →</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}


function detailHref(id: string, keyword: string, marketplace: string) {
  const params = new URLSearchParams({ q: keyword, mp: marketplace });
  return `/pencarian/${id}?${params.toString()}`;
}
