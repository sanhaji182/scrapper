export function formatRupiah(n: number): string {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(n || 0);
}

export const formatCurrency = formatRupiah;

export function formatRupiahShort(n: number): string {
  if (n >= 1_000_000_000) return `Rp ${(n / 1_000_000_000).toFixed(1)}M`;
  if (n >= 1_000_000) return `Rp ${(n / 1_000_000).toFixed(1)}jt`;
  if (n >= 1_000) return `Rp ${(n / 1_000).toFixed(0)}rb`;
  return `Rp ${n || 0}`;
}

export function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}jt`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}rb`;
  return String(n || 0);
}

export function formatDate(iso?: string | null): string {
  if (!iso) return "-";
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso));
}

export function formatDuration(startedAt?: string | null, finishedAt?: string | null): string {
  if (!startedAt || !finishedAt) return "-";
  const sec = Math.max(0, Math.round((new Date(finishedAt).getTime() - new Date(startedAt).getTime()) / 1000));
  if (sec < 60) return `${sec} detik`;
  return `${Math.floor(sec / 60)} menit ${sec % 60} detik`;
}

export function humanStatus(status: string): string {
  const map: Record<string, string> = {
    QUEUED: "Menunggu",
    RUNNING: "Sedang mencari",
    SUCCEEDED: "Selesai",
    FAILED: "Gagal",
    TIMED_OUT: "Waktu habis",
  };
  return map[status] ?? status;
}

export function marketplaceClass(mp: string): string {
  const map: Record<string, string> = {
    tokopedia: "mp-tokopedia",
    shopee: "mp-shopee",
    blibli: "mp-blibli",
    lazada: "mp-lazada",
  };
  return map[mp?.toLowerCase()] ?? "mp-default";
}

export function marketplaceLabel(mp: string): string {
  const map: Record<string, string> = {
    tokopedia: "Tokopedia",
    shopee: "Shopee",
    blibli: "Blibli",
    lazada: "Lazada",
  };
  return map[mp?.toLowerCase()] ?? mp;
}

export function runKeyword(run: { keyword?: string; input?: Record<string, unknown>; id: string }): string {
  const keyword = run.keyword ?? run.input?.keyword;
  return typeof keyword === "string" && keyword.trim() ? keyword : run.id.slice(0, 8);
}
