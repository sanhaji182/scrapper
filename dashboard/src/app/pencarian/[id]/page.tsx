import Link from "next/link";
import { redirect } from "next/navigation";
import { api } from "@/lib/api";
import { formatDate, marketplaceLabel, runKeyword } from "@/lib/format";
import { RunDetailClient } from "@/components/run-detail/RunDetailClient";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { RecoveringRunClient } from "@/components/run-detail/RecoveringRunClient";

export const dynamic = "force-dynamic";

export default async function PencarianDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ q?: string; mp?: string }>;
}) {
  const { id } = await params;
  const query = await searchParams;
  const [run, recent] = await Promise.all([
    api.getRun(id).catch(() => null),
    api.getRuns(100, 0).catch(() => null),
  ]);
  const validRuns = recent?.runs?.filter((item) => item.status === "SUCCEEDED" && item.item_count > 0) ?? [];
  const replacement = findReplacementRun(validRuns, query.q, query.mp, id);

  if (run) {
    if ((run.status === "FAILED" || run.item_count === 0) && replacement) {
      redirect(detailHref(replacement.id, query.q ?? runKeyword(replacement), query.mp ?? replacement.marketplace));
    }
    return <RunDetailClient initialRun={run} />;
  }

  if (replacement) redirect(detailHref(replacement.id, query.q ?? runKeyword(replacement), query.mp ?? replacement.marketplace));
  if (validRuns[0]) redirect(detailHref(validRuns[0].id, runKeyword(validRuns[0]), validRuns[0].marketplace));

  if (query.q || query.mp) return <RecoveringRunClient keyword={query.q} marketplace={query.mp} />;

  return (
    <div className="shell-container" style={{ display: "grid", gap: 18 }}>
      {validRuns.length > 0 ? (
        <section style={{ display: "grid", gap: 10 }}>
          <p className="sidebar-section-label" style={{ padding: 0 }}>Hasil valid terbaru</p>
          <div style={{ display: "grid", gap: 10 }}>
            {validRuns.map((item) => (
              <Link key={item.id} href={detailHref(item.id, runKeyword(item), item.marketplace)} className="soft-card" style={{ borderRadius: 20, padding: 15, textDecoration: "none", display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
                <div>
                  <h2 style={{ margin: 0, fontSize: 16, fontWeight: 900 }}>&quot;{runKeyword(item)}&quot;</h2>
                  <p className="text-muted" style={{ margin: "6px 0 0", fontSize: 12 }}>{marketplaceLabel(item.marketplace)} • {item.item_count} produk • {formatDate(item.created_at)}</p>
                </div>
                <StatusBadge status={item.status} />
              </Link>
            ))}
          </div>
        </section>
      ) : (
        <RecoveringRunClient keyword={query.q} marketplace={query.mp} />
      )}
    </div>
  );
}

function findReplacementRun(runs: NonNullable<Awaited<ReturnType<typeof api.getRuns>>["runs"]>, keyword?: string, marketplace?: string, currentID?: string) {
  const cleanKeyword = keyword?.trim().toLowerCase();
  const cleanMarketplace = marketplace?.trim().toLowerCase();
  if (!cleanKeyword && !cleanMarketplace) return null;
  return runs.find((item) => {
    if (item.id === currentID) return false;
    if (cleanMarketplace && item.marketplace?.toLowerCase() !== cleanMarketplace) return false;
    if (cleanKeyword && runKeyword(item).trim().toLowerCase() !== cleanKeyword) return false;
    return true;
  }) ?? null;
}

function detailHref(id: string, keyword?: string, marketplace?: string) {
  const params = new URLSearchParams();
  if (keyword) params.set("q", keyword);
  if (marketplace) params.set("mp", marketplace);
  const suffix = params.toString();
  return suffix ? `/pencarian/${id}?${suffix}` : `/pencarian/${id}`;
}
