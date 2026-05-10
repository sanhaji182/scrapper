import { api } from "@/lib/api";
import { HistoryClient } from "@/components/history/HistoryClient";

export const dynamic = "force-dynamic";

export default async function RiwayatPage() {
  const data = await api.getRuns(50, 0).catch(() => ({ runs: [], total: 0, limit: 50, offset: 0 }));
  return <HistoryClient initialRuns={data.runs} total={data.total} />;
}
