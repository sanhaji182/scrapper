"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { runKeyword } from "@/lib/format";
import type { Run } from "@/lib/types";

export function RecoveringRunClient({ keyword, marketplace }: { keyword?: string; marketplace?: string }) {
  const router = useRouter();
  const [message, setMessage] = useState("Mengecek hasil terbaru dari worker...");

  useEffect(() => {
    let cancelled = false;
    const cleanKeyword = keyword?.trim().toLowerCase();
    const cleanMarketplace = marketplace?.trim().toLowerCase();

    async function recover() {
      for (let attempt = 0; attempt < 20; attempt++) {
        const data = await api.getRuns(100, 0).catch(() => null);
        if (cancelled) return;
        const runs = data?.runs ?? [];
        const exact = findValidRun(runs, cleanKeyword, cleanMarketplace);
        const latest = runs.find((run) => run.status === "SUCCEEDED" && run.item_count > 0);
        const target = exact ?? latest;
        if (target) {
          const params = new URLSearchParams({ q: runKeyword(target), mp: target.marketplace });
          router.replace(`/pencarian/${target.id}?${params.toString()}`);
          return;
        }
        setMessage(attempt < 4 ? "Worker masih menyiapkan hasil..." : "Masih menunggu hasil valid. Ini biasanya selesai beberapa detik lagi.");
        await new Promise((resolve) => setTimeout(resolve, 1500));
      }
      setMessage("Belum ada hasil sukses yang bisa dibuka. Coba buka Riwayat atau mulai pencarian baru.");
    }

    void recover();
    return () => { cancelled = true; };
  }, [keyword, marketplace, router]);

  return (
    <div className="shell-container" style={{ display: "grid", gap: 18 }}>
      <div className="soft-card" style={{ borderRadius: 30, padding: "clamp(22px, 4vw, 38px)" }}>
        <span className="hero-eyebrow">Memulihkan pencarian</span>
        <h1 style={{ margin: "18px 0 8px", fontSize: "clamp(30px, 5vw, 58px)", lineHeight: .95, fontWeight: 950, letterSpacing: "-.07em" }}>
          Sedang mengambil hasil terbaru.
        </h1>
        <p className="hero-copy">{message}</p>
        <div style={{ display: "flex", gap: 10, flexWrap: "wrap", marginTop: 18 }}>
          <Link href="/riwayat" className="primary-button" style={{ display: "inline-flex", padding: "11px 16px" }}>Lihat riwayat</Link>
          <Link href="/" className="ghost-button" style={{ display: "inline-flex", padding: "11px 16px", fontWeight: 850 }}>Cari ulang</Link>
        </div>
      </div>
    </div>
  );
}

function findValidRun(runs: Run[], keyword?: string, marketplace?: string) {
  return runs.find((run) => {
    if (run.status !== "SUCCEEDED" || run.item_count <= 0) return false;
    if (marketplace && run.marketplace?.toLowerCase() !== marketplace) return false;
    if (keyword && runKeyword(run).trim().toLowerCase() !== keyword) return false;
    return true;
  }) ?? null;
}
