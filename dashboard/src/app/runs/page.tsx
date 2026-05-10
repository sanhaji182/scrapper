import Link from "next/link";
import { api } from "@/lib/api";
import type { RunsResponse } from "@/lib/types";
import { formatDate } from "@/lib/format";
import { statusClassName } from "@/lib/status";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";

export const dynamic = "force-dynamic";

export default async function RunsPage() {
  const data = await api.request<RunsResponse>("/v1/runs?limit=50&offset=0");

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Runs</h1>
          <p className="text-sm text-muted">{data.total} total scraping runs</p>
        </div>
        <Link
          href="/new-job"
          className="inline-flex items-center rounded-md primary-button px-3 py-1.5 text-sm"
        >
          New Job
        </Link>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="border-b table-head">
            <tr className="text-left text-muted">
              <th className="px-4 py-3">Run ID</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Marketplace</th>
              <th className="px-4 py-3">Items</th>
              <th className="px-4 py-3">Created</th>
            </tr>
          </thead>
          <tbody>
            {data.runs.map((run) => (
              <tr key={run.id} className="table-row">
                <td className="px-4 py-3 font-mono text-xs">
                  <Link href={`/runs/${run.id}`} className="text-[var(--foreground)] hover:text-[var(--accent)]">
                    {run.id}
                  </Link>
                </td>
                <td className="px-4 py-3">
                  <Badge className={statusClassName(run.status)}>{run.status}</Badge>
                </td>
                <td className="px-4 py-3 text-xs text-muted">{run.marketplace}</td>
                <td className="px-4 py-3 text-xs">{run.item_count}</td>
                <td className="px-4 py-3 text-xs text-muted">{formatDate(run.created_at)}</td>
              </tr>
            ))}
            {data.runs.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-10 text-center text-sm text-faint">
                  No runs yet. Submit a new scraping job to get started.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
