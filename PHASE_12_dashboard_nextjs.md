# PHASE 12 — Next.js Dashboard (Runs, Groups, AI Insights)

## Prerequisite
- Backend API Phase 1–11 sudah berjalan (scraper, normalize, ai-summary).
- Endpoint yang tersedia:
  - `GET  /v1/runs`
  - `GET  /v1/runs/:id`
  - `POST /v1/scrape/tokopedia/search`
  - `POST /v1/runs/:id/normalize`
  - `GET  /v1/runs/:id/normalized`
  - `POST /v1/runs/:id/ai-summary`
  - `GET  /v1/runs/:id/ai-summary`

## Objective
Membangun dashboard web modern untuk:
- Melihat list runs
- Melihat detail run (Products, Groups, AI Insights)
- Submit job baru

Dengan desain yang clean, data-first, dan cocok untuk developer tool.

## Stack
- Next.js 14 (App Router) + TypeScript
- Tailwind CSS
- shadcn/ui (komponen UI)

---

## Step 12.1 — Setup Project Frontend

Buat project baru (di repo terpisah atau folder `dashboard/`):

```bash
npx create-next-app@latest dashboard   --typescript   --tailwind   --eslint   --app

cd dashboard
```

Install shadcn/ui:

```bash
npx shadcn-ui@latest init
npx shadcn-ui@latest add button card input table badge tabs textarea skeleton
```

Atur base URL API backend di `.env.local`:

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

Buat helper fetcher:

**File:** `lib/api.ts`

```ts
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    // cache: "no-store" untuk data yang sering berubah
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`API error ${res.status}`);
  }

  return res.json();
}

export const api = { request };
```

---

## Step 12.2 — Definisikan Types Mirror Backend

**File:** `lib/types.ts`

```ts
export type RunStatus = "QUEUED" | "RUNNING" | "SUCCEEDED" | "FAILED" | "TIMED_OUT";

export interface Run {
  id: string;
  status: RunStatus;
  marketplace: string;
  input: unknown;
  item_count: number;
  created_at: string;
  started_at?: string | null;
  finished_at?: string | null;
}

export interface Product {
  id: string;
  name: string;
  price: number;
  original_price: number;
  discount_percent: number;
  rating: number;
  count_review: number;
  sold: number;
  url: string;
  image_url: string;
  shop_name: string;
  shop_city: string;
  is_official_store: boolean;
  marketplace: string;
}

export interface GroupedItem {
  product_id: string;
  marketplace: string;
  name: string;
  price: number;
  original_price: number;
  discount_percent: number;
  rating: number;
  count_review: number;
  shop_name: string;
  is_official_store: boolean;
  url: string;
}

export interface ProductGroup {
  group_id: string;
  canonical_name: string;
  brand: string;
  model: string;
  variant: string;
  category_path: string;
  important_specs: string[];
  items: GroupedItem[];
  min_price: number;
  max_price: number;
  avg_price: number;
  best_price_id: string;
}

export interface AISummaryResult {
  summary_text: string;
  recommended_items: {
    group_id: string;
    product_id: string;
    reason: string;
  }[];
}
```

---

## Step 12.3 — Layout & Navigation

Gunakan layout dengan sidebar kiri + content kanan.

**File:** `app/layout.tsx`

- Tambahkan font Inter
- Buat layout dasar dengan sidebar sederhana (`Runs`, `New Job`)

```tsx
import "./globals.css";
import type { Metadata } from "next";
import { Inter } from "next/font/google";
import Link from "next/link";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Tokopedia Scraper Dashboard",
  description: "Internal dashboard for scraping and AI insights",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={inter.className + " bg-slate-950 text-slate-100"}>
        <div className="flex min-h-screen">
          <aside className="w-64 border-r border-slate-800 bg-slate-950/80 p-4 flex flex-col gap-4">
            <div className="text-lg font-semibold tracking-tight">Scraper Console</div>
            <nav className="flex flex-col gap-2 text-sm">
              <Link href="/runs" className="hover:text-emerald-400">Runs</Link>
              <Link href="/new-job" className="hover:text-emerald-400">New Job</Link>
            </nav>
            <div className="mt-auto text-xs text-slate-500">Tokopedia Scraper • Internal</div>
          </aside>
          <main className="flex-1 p-6 overflow-auto bg-slate-900/80">
            {children}
          </main>
        </div>
      </body>
    </html>
  );
}
```

**File:** `app/page.tsx`

Redirect sederhana ke `/runs` atau tampilkan link.

---

## Step 12.4 — Runs List Page

**File:** `app/runs/page.tsx`

Gunakan server component untuk fetch list runs.

```tsx
import { api } from "@/lib/api";
import { Run } from "@/lib/types";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";

interface RunsResponse {
  runs: Run[];
  total: number;
  limit: number;
  offset: number;
}

function statusColor(status: string) {
  switch (status) {
    case "SUCCEEDED":
      return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30";
    case "RUNNING":
      return "bg-sky-500/10 text-sky-400 border-sky-500/30";
    case "FAILED":
      return "bg-red-500/10 text-red-400 border-red-500/30";
    default:
      return "bg-slate-500/10 text-slate-300 border-slate-500/30";
  }
}

export default async function RunsPage() {
  const data = await api.request<RunsResponse>("/v1/runs?limit=50&offset=0");

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight">Runs</h1>
        <Link
          href="/new-job"
          className="inline-flex items-center rounded-md bg-emerald-500 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-emerald-400"
        >
          New Job
        </Link>
      </div>
      <div className="rounded-lg border border-slate-800 bg-slate-950/50">
        <table className="w-full text-sm">
          <thead className="border-b border-slate-800 bg-slate-900/60">
            <tr className="text-left">
              <th className="px-4 py-2">Run ID</th>
              <th className="px-4 py-2">Status</th>
              <th className="px-4 py-2">Marketplace</th>
              <th className="px-4 py-2">Items</th>
              <th className="px-4 py-2">Created</th>
            </tr>
          </thead>
          <tbody>
            {data.runs.map((run) => (
              <tr key={run.id} className="border-t border-slate-800/60 hover:bg-slate-900/60">
                <td className="px-4 py-2 font-mono text-xs">
                  <Link href={`/runs/${run.id}`} className="hover:text-emerald-400">
                    {run.id}
                  </Link>
                </td>
                <td className="px-4 py-2">
                  <Badge className={statusColor(run.status)}>{run.status}</Badge>
                </td>
                <td className="px-4 py-2 text-xs text-slate-300">{run.marketplace}</td>
                <td className="px-4 py-2 text-xs">{run.item_count}</td>
                <td className="px-4 py-2 text-xs text-slate-400">
                  {new Date(run.created_at).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

---

## Step 12.5 — Run Detail Page (Tabs: Products, Groups, AI Insights)

**File:** `app/runs/[id]/page.tsx`

Struktur:
- Ambil `Run` detail dari `/v1/runs/:id`.
- Coba fetch `normalized` & `ai-summary` (boleh pakai `try/catch`).
- Render header + Tabs untuk 3 view.

Untuk singkat, di fase ini cukup:
- Tab `Products`: tampilkan data `result` jika `status == SUCCEEDED` (bisa call endpoint `/v1/runs/:id` dan gunakan field `result`).
- Tab `Groups`: pakai `GET /v1/runs/:id/normalized`.
- Tab `AI Insights`: pakai `GET /v1/runs/:id/ai-summary`, dan tombol untuk trigger `POST` jika belum ada.

> Di fase ini, kamu boleh membuat placeholder UI yang minimal; fokus pada plumbing API dan struktur tab.

---

## Step 12.6 — New Job Page

**File:** `app/new-job/page.tsx`

Form (client component):
- input keyword
- input max_items
- select sort_by
- optional min_price / max_price

On submit:
- `POST /v1/scrape/tokopedia/search`
- Redirect ke `/runs/[id]` berdasarkan `run_id` di response.

Gunakan Tailwind + shadcn/input/button untuk form.

---

## Step 12.7 — Verification

1. Jalankan backend (`make dev`) dan frontend (`npm run dev`) bersamaan.
2. Buka `http://localhost:3000/runs`:
   - Harus tampil list runs dari backend.
3. Klik salah satu run:
   - Halaman detail muncul, status benar.
4. Buka `http://localhost:3000/new-job` dan submit job:
   - Seharusnya redirect ke halaman detail run baru.

Jika semua ok, update checklist di AGENTS.md: Phase 12 ✅
