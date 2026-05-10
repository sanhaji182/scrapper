import Link from "next/link";
import { notFound } from "next/navigation";
import { api } from "@/lib/api";
import { formatCurrency, formatDate } from "@/lib/format";
import { statusClassName } from "@/lib/status";
import type { AISummaryResult, ProductGroup, Run } from "@/lib/types";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { normalizeRun, generateSummary } from "./actions";

export const dynamic = "force-dynamic";

async function optionalRequest<T>(path: string): Promise<T | null> {
  try {
    return await api.request<T>(path);
  } catch {
    return null;
  }
}

async function getRun(id: string): Promise<Run> {
  try {
    return await api.request<Run>(`/v1/runs/${id}`);
  } catch {
    notFound();
  }
}

export default async function RunDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [run, groups, summary] = await Promise.all([
    getRun(id),
    optionalRequest<ProductGroup[]>(`/v1/runs/${id}/normalized`),
    optionalRequest<AISummaryResult>(`/v1/runs/${id}/ai-summary`),
  ]);
  const products = run.result ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <Link href="/runs" className="text-sm text-muted hover:text-[var(--accent)]">
            ← Back to runs
          </Link>
          <h1 className="font-mono text-xl font-semibold tracking-tight">{run.id}</h1>
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted">
            <Badge className={statusClassName(run.status)}>{run.status}</Badge>
            <span>{run.marketplace}</span>
            <span>•</span>
            <span>{run.item_count} items</span>
            <span>•</span>
            <span>{formatDate(run.created_at)}</span>
          </div>
        </div>
        <form action={normalizeRun}>
          <input type="hidden" name="id" value={run.id} />
          <button className="ghost-button px-3 py-2 text-sm">
            Normalize
          </button>
        </form>
      </div>

      {run.error_message ? <Card className="p-4 text-sm text-[var(--danger)]">{run.error_message}</Card> : null}

      <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <Card className="p-4">
          <h2 className="mb-3 text-lg font-semibold">Products</h2>
          <div className="max-h-[560px] space-y-3 overflow-auto pr-1">
            {products.length > 0 ? (
              products.map((product) => (
                <div key={product.id} className="rounded-lg soft-panel p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <a href={product.url} target="_blank" className="font-medium text-[var(--foreground)] hover:text-[var(--accent)]">
                        {product.name}
                      </a>
                      <div className="mt-1 text-xs text-muted">
                        {product.shop_name} • {product.shop_city} • sold {product.sold}
                      </div>
                    </div>
                    <div className="whitespace-nowrap text-sm font-semibold accent-text">
                      {formatCurrency(product.price)}
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <p className="py-8 text-center text-sm text-faint">Products are available after the run succeeds.</p>
            )}
          </div>
        </Card>

        <div className="space-y-4">
          <Card className="p-4">
            <h2 className="mb-3 text-lg font-semibold">Groups</h2>
            <div className="space-y-3">
              {groups && groups.length > 0 ? (
                groups.map((group) => (
                  <div key={group.group_id} className="rounded-lg soft-panel p-3">
                    <div className="font-medium">{group.canonical_name || group.group_id}</div>
                    <div className="mt-1 text-xs text-muted">
                      {group.items.length} listings • min {formatCurrency(group.min_price)} • avg {formatCurrency(group.avg_price)}
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-sm text-faint">No normalized groups yet. Click Normalize to trigger the flow.</p>
              )}
            </div>
          </Card>

          <Card className="p-4">
            <h2 className="mb-3 text-lg font-semibold">AI Insights</h2>
            {summary ? (
              <div className="space-y-3 text-sm">
                <p className="text-muted">{summary.summary_text}</p>
                {summary.recommended_items.map((item) => (
                  <div key={`${item.group_id}-${item.product_id}`} className="rounded-lg soft-panel p-3">
                    <div className="font-mono text-xs accent-text">{item.product_id}</div>
                    <p className="mt-1 text-muted">{item.reason}</p>
                  </div>
                ))}
              </div>
            ) : (
              <form action={generateSummary} className="space-y-3">
                <input type="hidden" name="id" value={run.id} />
                <Textarea name="prompt" rows={4} placeholder="Ringkas pilihan terbaik untuk budget 10-12 juta" />
                <button className="rounded-md primary-button px-3 py-2 text-sm">
                  Generate AI Summary
                </button>
                <p className="text-xs text-faint">Uses the configured OpenAI-compatible provider from your API environment.</p>
              </form>
            )}
          </Card>
        </div>
      </section>
    </div>
  );
}
