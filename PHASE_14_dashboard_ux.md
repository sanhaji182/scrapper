# PHASE 14 — Dashboard UI/UX: Hybrid (Teknis + Non-Teknis)

## Prerequisite
- Phase 12 (Next.js scaffold) sudah ada.
- Backend API Phase 1–13 berjalan di `http://localhost:8080`.
- Node.js 20+, `npm` tersedia.

## Objective
Membangun ulang dashboard menjadi antarmuka **hybrid** yang:
- Bisa dipakai pengguna non-teknis (pembeli, analis bisnis) maupun developer
- Menampilkan info produk selengkap mungkin dari marketplace manapun
- Terasa hidup: micro-interactions, polling realtime, AI insights terintegrasi
- Tidak kaku: layout tiga kolom, kartu produk kaya info, animasi kontekstual

## Stack Tambahan
```bash
npm install framer-motion @radix-ui/react-slider @radix-ui/react-collapsible
npx shadcn-ui@latest add badge card tabs separator skeleton collapsible slider drawer
```

---

## Design Tokens

**File:** `app/globals.css` — ganti seluruh isi dengan yang berikut (setelah import Tailwind):

```css
@import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap');

:root {
  --bg:            #0d0d0f;
  --surface:       #141416;
  --surface-2:     #1a1a1d;
  --surface-3:     #202024;
  --surface-hover: #252529;
  --border:        rgba(255,255,255,0.07);
  --border-hover:  rgba(255,255,255,0.13);
  --text:          #e8e8ea;
  --text-muted:    #888890;
  --text-faint:    #3e3e45;
  --accent:        #22c55e;
  --accent-dim:    rgba(34,197,94,0.10);
  --accent-glow:   rgba(34,197,94,0.20);
  --warn:          #f59e0b;
  --error:         #ef4444;
  --radius-sm:     6px;
  --radius-md:     10px;
  --radius-lg:     14px;
  --radius-xl:     20px;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

html { scroll-behavior: smooth; }

body {
  font-family: 'Plus Jakarta Sans', sans-serif;
  background: var(--bg);
  color: var(--text);
  font-size: 14px;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
}

/* ── Marketplace Badge Colors ── */
.mp-tokopedia  { --mp-color: #22c55e; --mp-bg: rgba(34,197,94,0.10); }
.mp-shopee     { --mp-color: #f97316; --mp-bg: rgba(249,115,22,0.10); }
.mp-blibli     { --mp-color: #3b82f6; --mp-bg: rgba(59,130,246,0.10); }
.mp-lazada     { --mp-color: #a855f7; --mp-bg: rgba(168,85,247,0.10); }
.mp-default    { --mp-color: #888890; --mp-bg: rgba(136,136,144,0.10); }

.marketplace-badge {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 2px 8px; border-radius: 99px;
  font-size: 11px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase;
  background: var(--mp-bg); color: var(--mp-color);
  border: 1px solid var(--mp-color);
  opacity: 0.9;
}

/* ── Status Badge ── */
.status-badge {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 3px 10px; border-radius: 99px;
  font-size: 12px; font-weight: 500;
}
.status-badge .dot {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
}
.status-succeeded { background: rgba(34,197,94,0.10);  color: #4ade80; }
.status-succeeded .dot { background: #22c55e; }
.status-running   { background: rgba(59,130,246,0.10); color: #60a5fa; }
.status-running   .dot { background: #3b82f6; animation: pulse 1.2s ease-in-out infinite; }
.status-queued    { background: rgba(245,158,11,0.10); color: #fcd34d; }
.status-queued    .dot { background: #f59e0b; animation: pulse 1.5s ease-in-out infinite; }
.status-failed    { background: rgba(239,68,68,0.10);  color: #f87171; }
.status-failed    .dot { background: #ef4444; }
.status-timed_out { background: rgba(249,115,22,0.10); color: #fb923c; }
.status-timed_out .dot { background: #f97316; }

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%       { opacity: 0.5; transform: scale(0.85); }
}

/* ── Skeleton shimmer ── */
@keyframes shimmer {
  0%   { background-position: -200% 0; }
  100% { background-position:  200% 0; }
}
.skeleton-box {
  background: linear-gradient(90deg,
    var(--surface-3) 25%, rgba(255,255,255,0.04) 50%, var(--surface-3) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.6s ease-in-out infinite;
  border-radius: var(--radius-sm);
}

/* ── Scrollbar ── */
::-webkit-scrollbar       { width: 4px; height: 4px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.12); border-radius: 4px; }
::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.22); }
```

---

## Step 14.1 — Types & Format Helpers

**File:** `lib/types.ts` — tambahkan/ganti menjadi:

```ts
export type RunStatus = 'QUEUED' | 'RUNNING' | 'SUCCEEDED' | 'FAILED' | 'TIMED_OUT'
export type Marketplace = 'tokopedia' | 'shopee' | 'blibli' | 'lazada' | string

export interface Run {
  id: string
  status: RunStatus
  marketplace: Marketplace
  keyword: string
  item_count: number
  created_at: string
  started_at?: string | null
  finished_at?: string | null
  input?: Record<string, unknown>
  error_message?: string | null
}

export interface Product {
  id: string
  name: string
  price: number
  original_price: number
  discount_percent: number
  rating: number
  count_review: number
  sold: number
  url: string
  image_url: string
  shop_name: string
  shop_city: string
  shop_url?: string
  is_official_store: boolean
  marketplace: Marketplace
  category?: string
  badge?: string         // mis: "Best Seller", "Star Seller"
}

export interface GroupedItem {
  product_id: string
  marketplace: Marketplace
  name: string
  price: number
  original_price: number
  discount_percent: number
  rating: number
  count_review: number
  sold: number
  shop_name: string
  shop_city: string
  is_official_store: boolean
  url: string
  image_url: string
}

export interface ProductGroup {
  group_id: string
  canonical_name: string
  brand: string
  model: string
  variant: string
  category_path: string
  important_specs: string[]
  items: GroupedItem[]
  min_price: number
  max_price: number
  avg_price: number
  best_price_id: string
}

export interface RecommendedItem {
  group_id: string
  product_id: string
  reason: string
}

export interface AISummaryResult {
  summary_text: string
  recommended_items: RecommendedItem[]
}
```

**File:** `lib/format.ts`

```ts
export function formatRupiah(n: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency', currency: 'IDR',
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).format(n)
}

export function formatRupiahShort(n: number): string {
  if (n >= 1_000_000_000) return `Rp ${(n / 1_000_000_000).toFixed(1)}M`
  if (n >= 1_000_000)     return `Rp ${(n / 1_000_000).toFixed(1)}jt`
  if (n >= 1_000)         return `Rp ${(n / 1_000).toFixed(0)}rb`
  return `Rp ${n}`
}

export function formatCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}jt`
  if (n >= 1_000)     return `${(n / 1_000).toFixed(1)}rb`
  return String(n)
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('id-ID', {
    day: 'numeric', month: 'long', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  }).format(new Date(iso))
}

export function formatDuration(startedAt?: string | null, finishedAt?: string | null): string {
  if (!startedAt || !finishedAt) return '-'
  const sec = Math.round((new Date(finishedAt).getTime() - new Date(startedAt).getTime()) / 1000)
  if (sec < 60) return `${sec} detik`
  return `${Math.round(sec / 60)} menit ${sec % 60} detik`
}

export function humanStatus(status: string): string {
  const map: Record<string, string> = {
    QUEUED:    'Menunggu...',
    RUNNING:   'Sedang mencari...',
    SUCCEEDED: 'Selesai',
    FAILED:    'Gagal',
    TIMED_OUT: 'Waktu habis',
  }
  return map[status] ?? status
}

export function marketplaceClass(mp: string): string {
  const map: Record<string, string> = {
    tokopedia: 'mp-tokopedia',
    shopee:    'mp-shopee',
    blibli:    'mp-blibli',
    lazada:    'mp-lazada',
  }
  return map[mp?.toLowerCase()] ?? 'mp-default'
}

export function marketplaceLabel(mp: string): string {
  const map: Record<string, string> = {
    tokopedia: 'Tokopedia',
    shopee:    'Shopee',
    blibli:    'Blibli',
    lazada:    'Lazada',
  }
  return map[mp?.toLowerCase()] ?? mp
}
```

**File:** `lib/api.ts`

```ts
export const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    cache: 'no-store',
  })
  if (!res.ok) throw new Error(`API ${res.status}: ${path}`)
  return res.json()
}

export const api = {
  getRuns:        (limit = 50, offset = 0) =>
    req<{ runs: import('./types').Run[]; total: number }>(`/v1/runs?limit=${limit}&offset=${offset}`),

  getRun:         (id: string) =>
    req<import('./types').Run & { result?: import('./types').Product[] }>(`/v1/runs/${id}`),

  submitSearch:   (body: object) =>
    req<{ run_id: string; status: string }>('/v1/scrape/tokopedia/search', {
      method: 'POST', body: JSON.stringify(body),
    }),

  getNormalized:  (id: string) =>
    req<import('./types').ProductGroup[]>(`/v1/runs/${id}/normalized`),

  triggerNormalize: (id: string) =>
    req<{ group_cnt: number }>(`/v1/runs/${id}/normalize`, { method: 'POST' }),

  getAISummary:   (id: string) =>
    req<import('./types').AISummaryResult>(`/v1/runs/${id}/ai-summary`),

  triggerAISummary: (id: string, prompt?: string) =>
    req<import('./types').AISummaryResult>(`/v1/runs/${id}/ai-summary`, {
      method: 'POST', body: JSON.stringify({ prompt: prompt ?? '' }),
    }),
}
```

---

## Step 14.2 — Layout & Navigation

**File:** `app/layout.tsx`

```tsx
import './globals.css'
import type { Metadata } from 'next'
import { Sidebar } from '@/components/layout/Sidebar'
import { BottomNav } from '@/components/layout/BottomNav'

export const metadata: Metadata = {
  title: 'PriceScope — Bandingkan Harga Marketplace',
  description: 'Cari dan bandingkan harga produk dari Tokopedia, Shopee, dan marketplace lainnya.',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id">
      <body>
        <div className="flex min-h-screen">
          <Sidebar />
          <main className="flex-1 min-w-0 pb-24 md:pb-0">
            {children}
          </main>
        </div>
        <BottomNav />
      </body>
    </html>
  )
}
```

**File:** `components/layout/Sidebar.tsx`

```tsx
'use client'
import Link from 'next/link'
import { usePathname } from 'next/navigation'

const NAV = [
  { href: '/',        label: 'Cari Produk',     icon: '⌕' },
  { href: '/riwayat', label: 'Riwayat',          icon: '◷' },
  { href: '/pengaturan', label: 'Pengaturan',    icon: '⚙' },
]

export function Sidebar() {
  const path = usePathname()
  return (
    <aside style={{
      width: 220,
      flexShrink: 0,
      borderRight: '1px solid var(--border)',
      background: 'var(--surface)',
      padding: '24px 0',
      display: 'flex',
      flexDirection: 'column',
      gap: 4,
      position: 'sticky',
      top: 0,
      height: '100vh',
    }} className="hidden md:flex">

      {/* Logo */}
      <div style={{ padding: '0 20px 20px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 16, fontWeight: 700, color: 'var(--accent)' }}>PriceScope</span>
        <span style={{ display: 'block', fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
          Bandingkan harga marketplace
        </span>
      </div>

      {/* Nav items */}
      <nav style={{ padding: '12px 10px', flex: 1 }}>
        {NAV.map(n => {
          const active = path === n.href || (n.href !== '/' && path.startsWith(n.href))
          return (
            <Link key={n.href} href={n.href} style={{
              display: 'flex', alignItems: 'center', gap: 10,
              padding: '9px 12px', borderRadius: 'var(--radius-md)',
              color: active ? 'var(--accent)' : 'var(--text-muted)',
              background: active ? 'var(--accent-dim)' : 'transparent',
              fontWeight: active ? 600 : 400,
              fontSize: 13.5,
              textDecoration: 'none',
              transition: 'all 0.15s',
            }}>
              <span style={{ fontSize: 16, opacity: active ? 1 : 0.6 }}>{n.icon}</span>
              {n.label}
            </Link>
          )
        })}
      </nav>
    </aside>
  )
}
```

**File:** `components/layout/BottomNav.tsx`

```tsx
'use client'
import Link from 'next/link'
import { usePathname } from 'next/navigation'

const NAV = [
  { href: '/',        label: 'Cari',     icon: '⌕' },
  { href: '/riwayat', label: 'Riwayat',  icon: '◷' },
  { href: '/pengaturan', label: 'Setelan', icon: '⚙' },
]

export function BottomNav() {
  const path = usePathname()
  return (
    <nav className="md:hidden" style={{
      position: 'fixed', bottom: 0, left: 0, right: 0,
      borderTop: '1px solid var(--border)',
      background: 'rgba(20,20,22,0.95)',
      backdropFilter: 'blur(12px)',
      display: 'flex', zIndex: 100,
    }}>
      {NAV.map(n => {
        const active = path === n.href || (n.href !== '/' && path.startsWith(n.href))
        return (
          <Link key={n.href} href={n.href} style={{
            flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center',
            gap: 3, padding: '10px 0 14px',
            color: active ? 'var(--accent)' : 'var(--text-muted)',
            textDecoration: 'none', fontSize: 10, fontWeight: active ? 600 : 400,
          }}>
            <span style={{ fontSize: 20 }}>{n.icon}</span>
            {n.label}
          </Link>
        )
      })}
    </nav>
  )
}
```

---

## Step 14.3 — Halaman Beranda (Search Hub)

**File:** `app/page.tsx`

```tsx
import { SearchHub } from '@/components/search/SearchHub'

export default function Home() {
  return <SearchHub />
}
```

**File:** `components/search/SearchHub.tsx`

```tsx
'use client'
import { useState, FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'

const SORT_OPTIONS = [
  { value: 'relevancy',  label: 'Paling Relevan' },
  { value: 'price_asc',  label: 'Harga Terendah' },
  { value: 'price_desc', label: 'Harga Tertinggi' },
  { value: 'latest',     label: 'Terbaru' },
]
const MAX_ITEMS_OPTIONS = [20, 50, 100, 200]
const RECENT_KEY = 'ps_recent_searches'

function getRecent(): string[] {
  if (typeof window === 'undefined') return []
  try { return JSON.parse(localStorage.getItem(RECENT_KEY) ?? '[]') } catch { return [] }
}

function saveRecent(keyword: string) {
  const prev = getRecent().filter(k => k !== keyword)
  localStorage.setItem(RECENT_KEY, JSON.stringify([keyword, ...prev].slice(0, 8)))
}

export function SearchHub() {
  const router = useRouter()
  const [keyword, setKeyword] = useState('')
  const [showFilter, setShowFilter] = useState(false)
  const [minPrice, setMinPrice] = useState('')
  const [maxPrice, setMaxPrice] = useState('')
  const [sortBy, setSortBy] = useState('relevancy')
  const [maxItems, setMaxItems] = useState(50)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const recent = typeof window !== 'undefined' ? getRecent() : []

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!keyword.trim()) return
    setLoading(true)
    setError('')
    try {
      const body: Record<string, unknown> = {
        keyword: keyword.trim(),
        maxItems,
        sortBy,
      }
      if (minPrice) body.minPrice = parseInt(minPrice.replace(/\D/g, ''), 10)
      if (maxPrice) body.maxPrice = parseInt(maxPrice.replace(/\D/g, ''), 10)

      const res = await api.submitSearch(body)
      saveRecent(keyword.trim())
      router.push(`/pencarian/${res.run_id}`)
    } catch (err: unknown) {
      setError('Gagal memulai pencarian. Pastikan server berjalan dan coba lagi.')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  function formatPriceInput(val: string): string {
    const digits = val.replace(/\D/g, '')
    return digits ? parseInt(digits).toLocaleString('id-ID') : ''
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex', flexDirection: 'column',
      alignItems: 'center', justifyContent: 'center',
      padding: '40px 24px',
    }}>

      {/* Subtle background */}
      <div style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 0,
        background: 'radial-gradient(ellipse 70% 40% at 50% 0%, rgba(34,197,94,0.04) 0%, transparent 70%)',
        pointerEvents: 'none',
      }} />

      <div style={{ position: 'relative', zIndex: 1, width: '100%', maxWidth: 580 }}>

        {/* Heading */}
        <div style={{ textAlign: 'center', marginBottom: 40 }}>
          <span style={{
            fontSize: 11, fontWeight: 600, letterSpacing: '0.1em', textTransform: 'uppercase',
            color: 'var(--accent)', display: 'block', marginBottom: 12,
          }}>PriceScope</span>
          <h1 style={{ fontSize: 'clamp(24px, 4vw, 34px)', fontWeight: 700, lineHeight: 1.25, marginBottom: 12 }}>
            Bandingkan harga dari<br />semua marketplace
          </h1>
          <p style={{ color: 'var(--text-muted)', fontSize: 15 }}>
            Cari produk sekali, dapatkan harga terbaik dari Tokopedia, Shopee, dan lainnya.
          </p>
        </div>

        {/* Search form */}
        <form onSubmit={handleSubmit}>
          <div style={{
            display: 'flex', gap: 10, alignItems: 'stretch',
            background: 'var(--surface-2)', border: '1px solid var(--border)',
            borderRadius: 'var(--radius-xl)', padding: 6,
            transition: 'border-color 0.2s',
          }}>
            <input
              autoFocus
              value={keyword}
              onChange={e => setKeyword(e.target.value)}
              placeholder="Cari produk, misalnya: laptop gaming, iphone 15..."
              disabled={loading}
              style={{
                flex: 1, background: 'transparent', border: 'none', outline: 'none',
                color: 'var(--text)', fontSize: 15, padding: '10px 14px',
              }}
            />
            <button
              type="submit"
              disabled={loading || !keyword.trim()}
              style={{
                background: loading ? 'rgba(34,197,94,0.6)' : 'var(--accent)',
                color: '#052e16', fontWeight: 600, fontSize: 14,
                border: 'none', borderRadius: 'var(--radius-lg)',
                padding: '10px 22px', cursor: loading ? 'wait' : 'pointer',
                transition: 'all 0.2s', whiteSpace: 'nowrap',
                opacity: !keyword.trim() ? 0.5 : 1,
              }}
            >
              {loading ? 'Mencari...' : 'Cari Sekarang'}
            </button>
          </div>

          {/* Filter toggle */}
          <button
            type="button"
            onClick={() => setShowFilter(v => !v)}
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              color: 'var(--text-muted)', fontSize: 13, marginTop: 12,
              display: 'flex', alignItems: 'center', gap: 6,
              padding: '4px 0',
            }}
          >
            <span style={{ transition: 'transform 0.2s', transform: showFilter ? 'rotate(180deg)' : 'none' }}>▾</span>
            Filter opsional
          </button>

          {showFilter && (
            <div style={{
              background: 'var(--surface-2)', border: '1px solid var(--border)',
              borderRadius: 'var(--radius-lg)', padding: 18, marginTop: 8,
              display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12,
            }}>
              <div>
                <label style={{ fontSize: 12, color: 'var(--text-muted)', display: 'block', marginBottom: 6 }}>
                  Harga Minimum
                </label>
                <div style={{ position: 'relative' }}>
                  <span style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', fontSize: 13 }}>Rp</span>
                  <input
                    value={minPrice}
                    onChange={e => setMinPrice(formatPriceInput(e.target.value))}
                    placeholder="0"
                    style={{
                      width: '100%', background: 'var(--surface-3)', border: '1px solid var(--border)',
                      borderRadius: 'var(--radius-sm)', color: 'var(--text)', fontSize: 13,
                      padding: '8px 10px 8px 32px', outline: 'none',
                    }}
                  />
                </div>
              </div>
              <div>
                <label style={{ fontSize: 12, color: 'var(--text-muted)', display: 'block', marginBottom: 6 }}>
                  Harga Maksimum
                </label>
                <div style={{ position: 'relative' }}>
                  <span style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)', fontSize: 13 }}>Rp</span>
                  <input
                    value={maxPrice}
                    onChange={e => setMaxPrice(formatPriceInput(e.target.value))}
                    placeholder="Semua"
                    style={{
                      width: '100%', background: 'var(--surface-3)', border: '1px solid var(--border)',
                      borderRadius: 'var(--radius-sm)', color: 'var(--text)', fontSize: 13,
                      padding: '8px 10px 8px 32px', outline: 'none',
                    }}
                  />
                </div>
              </div>
              <div>
                <label style={{ fontSize: 12, color: 'var(--text-muted)', display: 'block', marginBottom: 6 }}>Urutan Hasil</label>
                <select
                  value={sortBy}
                  onChange={e => setSortBy(e.target.value)}
                  style={{
                    width: '100%', background: 'var(--surface-3)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-sm)', color: 'var(--text)', fontSize: 13,
                    padding: '8px 10px', outline: 'none',
                  }}
                >
                  {SORT_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
                </select>
              </div>
              <div>
                <label style={{ fontSize: 12, color: 'var(--text-muted)', display: 'block', marginBottom: 6 }}>Jumlah Produk</label>
                <select
                  value={maxItems}
                  onChange={e => setMaxItems(Number(e.target.value))}
                  style={{
                    width: '100%', background: 'var(--surface-3)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-sm)', color: 'var(--text)', fontSize: 13,
                    padding: '8px 10px', outline: 'none',
                  }}
                >
                  {MAX_ITEMS_OPTIONS.map(n => <option key={n} value={n}>{n} produk</option>)}
                </select>
              </div>
            </div>
          )}

          {error && (
            <p style={{ color: 'var(--error)', fontSize: 13, marginTop: 10 }}>{error}</p>
          )}
        </form>

        {/* Recent searches */}
        {recent.length > 0 && (
          <div style={{ marginTop: 28 }}>
            <p style={{ fontSize: 12, color: 'var(--text-faint)', marginBottom: 10, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
              Pencarian terakhir
            </p>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              {recent.map(kw => (
                <button
                  key={kw}
                  type="button"
                  onClick={() => { setKeyword(kw) }}
                  style={{
                    background: 'var(--surface-2)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-xl)', color: 'var(--text-muted)', fontSize: 13,
                    padding: '5px 14px', cursor: 'pointer', transition: 'all 0.15s',
                  }}
                  onMouseEnter={e => {
                    (e.target as HTMLElement).style.borderColor = 'var(--border-hover)'
                    ;(e.target as HTMLElement).style.color = 'var(--text)'
                  }}
                  onMouseLeave={e => {
                    (e.target as HTMLElement).style.borderColor = 'var(--border)'
                    ;(e.target as HTMLElement).style.color = 'var(--text-muted)'
                  }}
                >
                  {kw}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
```

---

## Step 14.4 — Halaman Detail Pencarian

**File:** `app/pencarian/[id]/page.tsx`

```tsx
import { api } from '@/lib/api'
import { RunDetailClient } from '@/components/run-detail/RunDetailClient'

export default async function PencarianDetailPage({ params }: { params: { id: string } }) {
  const run = await api.getRun(params.id).catch(() => null)
  if (!run) return (
    <div style={{ padding: 40, color: 'var(--text-muted)' }}>
      Pencarian tidak ditemukan. <a href="/riwayat" style={{ color: 'var(--accent)' }}>Kembali ke riwayat</a>
    </div>
  )
  return <RunDetailClient initialRun={run} />
}
```

**File:** `components/run-detail/RunDetailClient.tsx`

```tsx
'use client'
import { useState, useEffect, useCallback } from 'react'
import Link from 'next/link'
import { api } from '@/lib/api'
import { Run, Product, ProductGroup, AISummaryResult } from '@/lib/types'
import { formatDate, formatDuration, humanStatus, marketplaceLabel } from '@/lib/format'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { ProductsGrid } from './ProductsGrid'
import { GroupsView } from './GroupsView'
import { AIInsightsPanel } from './AIInsightsPanel'
import { RunTechDetails } from './RunTechDetails'

type Tab = 'produk' | 'kelompok' | 'ai' | 'teknis'

export function RunDetailClient({ initialRun }: { initialRun: Run & { result?: Product[] } }) {
  const [run, setRun] = useState(initialRun)
  const [products, setProducts] = useState<Product[]>(initialRun.result ?? [])
  const [groups, setGroups] = useState<ProductGroup[]>([])
  const [aiSummary, setAiSummary] = useState<AISummaryResult | null>(null)
  const [aiStatus, setAiStatus] = useState<'idle' | 'loading' | 'done' | 'error'>('idle')
  const [activeTab, setActiveTab] = useState<Tab>('produk')

  // Polling saat masih running
  useEffect(() => {
    if (run.status !== 'RUNNING' && run.status !== 'QUEUED') return
    const interval = setInterval(async () => {
      const updated = await api.getRun(run.id).catch(() => null)
      if (!updated) return
      setRun(updated)
      if (updated.result) setProducts(updated.result)
    }, 3000)
    return () => clearInterval(interval)
  }, [run.id, run.status])

  // Load groups & AI saat SUCCEEDED
  const loadAI = useCallback(async () => {
    setAiStatus('loading')
    try {
      // Coba get normalized dulu
      let g: ProductGroup[] = []
      try { g = await api.getNormalized(run.id) } catch {
        await api.triggerNormalize(run.id)
        g = await api.getNormalized(run.id)
      }
      setGroups(g)

      // Coba get AI summary dulu
      let summary: AISummaryResult | null = null
      try { summary = await api.getAISummary(run.id) } catch {
        summary = await api.triggerAISummary(run.id)
      }
      setAiSummary(summary)
      setAiStatus('done')
    } catch {
      setAiStatus('error')
    }
  }, [run.id])

  useEffect(() => {
    if (run.status === 'SUCCEEDED' && aiStatus === 'idle') loadAI()
  }, [run.status, aiStatus, loadAI])

  const keyword = (run.input as Record<string, string>)?.keyword ?? run.id

  const TABS: { id: Tab; label: string; badge?: string }[] = [
    { id: 'produk',   label: 'Semua Produk',    badge: run.item_count > 0 ? String(run.item_count) : undefined },
    { id: 'kelompok', label: 'Kelompok Serupa', badge: groups.length > 0 ? String(groups.length) : undefined },
    { id: 'ai',       label: '🤖 Rekomendasi AI', badge: aiSummary?.recommended_items?.length ? String(aiSummary.recommended_items.length) : undefined },
    { id: 'teknis',   label: 'Detail Teknis' },
  ]

  return (
    <div style={{ maxWidth: 1280, margin: '0 auto', padding: '24px 20px' }}>

      {/* Header */}
      <div style={{ marginBottom: 24 }}>
        <Link href="/riwayat" style={{ color: 'var(--text-muted)', fontSize: 13, textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: 6, marginBottom: 16 }}>
          ← Kembali ke Riwayat
        </Link>

        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, marginBottom: 6 }}>
              Hasil: "{keyword}"
            </h1>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap', color: 'var(--text-muted)', fontSize: 13 }}>
              <StatusBadge status={run.status} />
              <span>{marketplaceLabel(run.marketplace)}</span>
              {run.item_count > 0 && <span>{run.item_count} produk ditemukan</span>}
              <span>{formatDate(run.created_at)}</span>
              {run.started_at && run.finished_at && (
                <span>Durasi: {formatDuration(run.started_at, run.finished_at)}</span>
              )}
            </div>
          </div>
          <Link href="/" style={{
            background: 'var(--accent-dim)', border: '1px solid rgba(34,197,94,0.2)',
            color: 'var(--accent)', borderRadius: 'var(--radius-md)',
            padding: '8px 16px', fontSize: 13, fontWeight: 500, textDecoration: 'none',
          }}>
            + Pencarian Baru
          </Link>
        </div>
      </div>

      {/* Running banner */}
      {(run.status === 'RUNNING' || run.status === 'QUEUED') && (
        <div style={{
          background: 'rgba(59,130,246,0.08)', border: '1px solid rgba(59,130,246,0.2)',
          borderRadius: 'var(--radius-md)', padding: '12px 16px', marginBottom: 20,
          display: 'flex', alignItems: 'center', gap: 10, fontSize: 13, color: '#93c5fd',
        }}>
          <span style={{ animation: 'pulse 1.2s ease-in-out infinite', display: 'inline-block' }}>⟳</span>
          {humanStatus(run.status)} — halaman ini diperbarui otomatis setiap 3 detik
          {run.item_count > 0 && <span style={{ marginLeft: 'auto', fontWeight: 600 }}>{run.item_count} produk sejauh ini</span>}
        </div>
      )}

      {/* Failed banner */}
      {run.status === 'FAILED' && (
        <div style={{
          background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.2)',
          borderRadius: 'var(--radius-md)', padding: '12px 16px', marginBottom: 20,
          fontSize: 13, color: '#fca5a5',
        }}>
          Pencarian gagal. {run.error_message ? `Pesan: ${run.error_message}` : 'Silakan coba pencarian baru.'}
        </div>
      )}

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 4, borderBottom: '1px solid var(--border)',
        marginBottom: 24, overflowX: 'auto',
      }}>
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setActiveTab(t.id)}
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              padding: '10px 16px',
              borderBottom: activeTab === t.id ? '2px solid var(--accent)' : '2px solid transparent',
              color: activeTab === t.id ? 'var(--text)' : 'var(--text-muted)',
              fontWeight: activeTab === t.id ? 600 : 400,
              fontSize: 13.5, whiteSpace: 'nowrap',
              display: 'flex', alignItems: 'center', gap: 7,
              transition: 'color 0.15s',
            }}
          >
            {t.label}
            {t.badge && (
              <span style={{
                background: activeTab === t.id ? 'var(--accent-dim)' : 'var(--surface-3)',
                color: activeTab === t.id ? 'var(--accent)' : 'var(--text-muted)',
                fontSize: 11, fontWeight: 600, padding: '1px 7px', borderRadius: 99,
              }}>{t.badge}</span>
            )}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === 'produk'   && <ProductsGrid products={products} runStatus={run.status} />}
      {activeTab === 'kelompok' && <GroupsView groups={groups} loading={aiStatus === 'loading'} />}
      {activeTab === 'ai'       && <AIInsightsPanel summary={aiSummary} status={aiStatus} products={products} />}
      {activeTab === 'teknis'   && <RunTechDetails run={run} />}
    </div>
  )
}
```

---

## Step 14.5 — ProductsGrid (Grid Produk Lengkap)

**File:** `components/run-detail/ProductsGrid.tsx`

```tsx
'use client'
import { useState, useMemo } from 'react'
import { Product, RunStatus } from '@/lib/types'
import { formatRupiah, formatCount, marketplaceClass, marketplaceLabel } from '@/lib/format'
import { ProductCardSkeleton } from '@/components/ui/ProductCardSkeleton'

type SortKey = 'price_asc' | 'price_desc' | 'rating' | 'sold' | 'discount'

interface Props { products: Product[]; runStatus: RunStatus }

export function ProductsGrid({ products, runStatus }: Props) {
  const [sortBy, setSortBy] = useState<SortKey>('price_asc')
  const [officialOnly, setOfficialOnly] = useState(false)
  const [search, setSearch] = useState('')
  const [minPrice, setMinPrice] = useState(0)
  const [maxPrice, setMaxPrice] = useState(0)

  const priceRange = useMemo(() => {
    if (!products.length) return { min: 0, max: 0 }
    return {
      min: Math.min(...products.map(p => p.price)),
      max: Math.max(...products.map(p => p.price)),
    }
  }, [products])

  const filtered = useMemo(() => {
    let list = [...products]
    if (officialOnly) list = list.filter(p => p.is_official_store)
    if (search) list = list.filter(p => p.name.toLowerCase().includes(search.toLowerCase()))
    if (minPrice > 0) list = list.filter(p => p.price >= minPrice)
    if (maxPrice > 0) list = list.filter(p => p.price <= maxPrice)
    switch (sortBy) {
      case 'price_asc':  return list.sort((a, b) => a.price - b.price)
      case 'price_desc': return list.sort((a, b) => b.price - a.price)
      case 'rating':     return list.sort((a, b) => b.rating - a.rating)
      case 'sold':       return list.sort((a, b) => b.sold - a.sold)
      case 'discount':   return list.sort((a, b) => b.discount_percent - a.discount_percent)
    }
  }, [products, officialOnly, search, sortBy, minPrice, maxPrice])

  const isLoading = (runStatus === 'RUNNING' || runStatus === 'QUEUED') && products.length === 0

  return (
    <div>
      {/* Toolbar */}
      <div style={{
        display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap',
        marginBottom: 18, padding: '12px 14px',
        background: 'var(--surface-2)', border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
      }}>
        <input
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="Filter nama produk..."
          style={{
            flex: 1, minWidth: 160, background: 'var(--surface-3)',
            border: '1px solid var(--border)', borderRadius: 'var(--radius-sm)',
            color: 'var(--text)', fontSize: 13, padding: '7px 12px', outline: 'none',
          }}
        />
        <select
          value={sortBy}
          onChange={e => setSortBy(e.target.value as SortKey)}
          style={{
            background: 'var(--surface-3)', border: '1px solid var(--border)',
            borderRadius: 'var(--radius-sm)', color: 'var(--text)', fontSize: 13,
            padding: '7px 10px', outline: 'none',
          }}
        >
          <option value="price_asc">Harga Terendah</option>
          <option value="price_desc">Harga Tertinggi</option>
          <option value="rating">Rating Tertinggi</option>
          <option value="sold">Terlaris</option>
          <option value="discount">Diskon Terbesar</option>
        </select>
        <label style={{ display: 'flex', alignItems: 'center', gap: 7, fontSize: 13, color: 'var(--text-muted)', cursor: 'pointer', whiteSpace: 'nowrap' }}>
          <input
            type="checkbox"
            checked={officialOnly}
            onChange={e => setOfficialOnly(e.target.checked)}
            style={{ accentColor: 'var(--accent)' }}
          />
          Toko Resmi
        </label>
        <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 'auto', whiteSpace: 'nowrap' }}>
          {filtered.length} dari {products.length} produk
        </span>
      </div>

      {/* Grid */}
      {isLoading ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 14 }}>
          {Array.from({ length: 8 }).map((_, i) => <ProductCardSkeleton key={i} />)}
        </div>
      ) : filtered.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '60px 20px', color: 'var(--text-muted)' }}>
          <div style={{ fontSize: 32, marginBottom: 12 }}>◎</div>
          <p>Tidak ada produk yang cocok dengan filter ini.</p>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 14 }}>
          {filtered.map(p => <ProductCard key={`${p.marketplace}-${p.id}`} product={p} />)}
        </div>
      )}
    </div>
  )
}

function ProductCard({ product: p }: { product: Product }) {
  const [imgErr, setImgErr] = useState(false)
  const mpClass = marketplaceClass(p.marketplace)

  return (
    <div style={{
      background: 'var(--surface-2)', border: '1px solid var(--border)',
      borderRadius: 'var(--radius-lg)', overflow: 'hidden',
      transition: 'transform 0.18s, border-color 0.18s, box-shadow 0.18s',
      display: 'flex', flexDirection: 'column',
    }}
    onMouseEnter={e => {
      const el = e.currentTarget
      el.style.transform = 'translateY(-2px) scale(1.015)'
      el.style.borderColor = 'var(--border-hover)'
      el.style.boxShadow = '0 8px 24px rgba(0,0,0,0.3)'
    }}
    onMouseLeave={e => {
      const el = e.currentTarget
      el.style.transform = 'none'
      el.style.borderColor = 'var(--border)'
      el.style.boxShadow = 'none'
    }}>

      {/* Image area */}
      <div style={{ position: 'relative', aspectRatio: '1', background: 'var(--surface-3)', overflow: 'hidden' }}>
        {p.image_url && !imgErr ? (
          <img
            src={p.image_url} alt={p.name}
            onError={() => setImgErr(true)}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-faint)', fontSize: 28 }}>◫</div>
        )}

        {/* Discount badge */}
        {p.discount_percent > 0 && (
          <span style={{
            position: 'absolute', top: 8, right: 8,
            background: 'var(--error)', color: '#fff',
            fontSize: 11, fontWeight: 700, padding: '2px 7px', borderRadius: 99,
          }}>-{p.discount_percent}%</span>
        )}

        {/* Marketplace badge */}
        <span className={`marketplace-badge ${mpClass}`} style={{ position: 'absolute', bottom: 8, left: 8 }}>
          {marketplaceLabel(p.marketplace)}
        </span>
      </div>

      {/* Content */}
      <div style={{ padding: '12px 12px 14px', display: 'flex', flexDirection: 'column', gap: 6, flex: 1 }}>

        <p style={{
          fontSize: 13, fontWeight: 500, lineHeight: 1.4,
          display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical',
          overflow: 'hidden', color: 'var(--text)',
        }}>{p.name}</p>

        {/* Rating */}
        {p.rating > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 12, color: 'var(--text-muted)' }}>
            <span style={{ color: '#facc15' }}>★</span>
            <span>{p.rating.toFixed(1)}</span>
            {p.count_review > 0 && <span>({formatCount(p.count_review)} ulasan)</span>}
          </div>
        )}

        {/* Sold */}
        {p.sold > 0 && (
          <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>
            Terjual {formatCount(p.sold)}
          </p>
        )}

        {/* Price */}
        <div style={{ marginTop: 'auto', paddingTop: 6 }}>
          <p style={{ fontSize: 16, fontWeight: 700, color: 'var(--text)' }}>
            {formatRupiah(p.price)}
          </p>
          {p.original_price > p.price && (
            <p style={{ fontSize: 12, color: 'var(--text-muted)', textDecoration: 'line-through' }}>
              {formatRupiah(p.original_price)}
            </p>
          )}
        </div>

        {/* Shop */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 12, color: 'var(--text-muted)', flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {p.shop_name}
          </span>
          {p.is_official_store && (
            <span style={{
              background: 'rgba(34,197,94,0.1)', color: 'var(--accent)',
              fontSize: 10, fontWeight: 600, padding: '1px 7px', borderRadius: 99,
              border: '1px solid rgba(34,197,94,0.2)', whiteSpace: 'nowrap',
            }}>✓ Resmi</span>
          )}
        </div>

        {p.shop_city && (
          <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>📍 {p.shop_city}</p>
        )}

        {/* CTA */}
        <a
          href={p.url} target="_blank" rel="noopener noreferrer"
          style={{
            display: 'block', textAlign: 'center', marginTop: 8,
            background: 'var(--surface-3)', border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)', padding: '8px', fontSize: 12, fontWeight: 500,
            color: 'var(--text)', textDecoration: 'none', transition: 'all 0.15s',
          }}
          onMouseEnter={e => {
            (e.currentTarget as HTMLAnchorElement).style.background = 'var(--accent-dim)'
            ;(e.currentTarget as HTMLAnchorElement).style.borderColor = 'rgba(34,197,94,0.3)'
            ;(e.currentTarget as HTMLAnchorElement).style.color = 'var(--accent)'
          }}
          onMouseLeave={e => {
            (e.currentTarget as HTMLAnchorElement).style.background = 'var(--surface-3)'
            ;(e.currentTarget as HTMLAnchorElement).style.borderColor = 'var(--border)'
            ;(e.currentTarget as HTMLAnchorElement).style.color = 'var(--text)'
          }}
        >
          Lihat di {marketplaceLabel(p.marketplace)} →
        </a>
      </div>
    </div>
  )
}
```

---

## Step 14.6 — GroupsView (Kelompok Serupa)

**File:** `components/run-detail/GroupsView.tsx`

```tsx
'use client'
import { useState } from 'react'
import { ProductGroup, GroupedItem } from '@/lib/types'
import { formatRupiah, formatRupiahShort, formatCount, marketplaceClass, marketplaceLabel } from '@/lib/format'

interface Props { groups: ProductGroup[]; loading: boolean }

export function GroupsView({ groups, loading }: Props) {
  if (loading) return (
    <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: 14 }}>
      <div style={{ fontSize: 28, marginBottom: 12, animation: 'pulse 1.2s ease-in-out infinite', display: 'inline-block' }}>◎</div>
      <p>Sedang mengelompokkan produk serupa...</p>
    </div>
  )

  if (!groups.length) return (
    <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
      <p>Belum ada kelompok produk. Tunggu analisis selesai.</p>
    </div>
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
        {groups.length} kelompok produk serupa ditemukan
      </p>
      {groups.map(g => <GroupCard key={g.group_id} group={g} />)}
    </div>
  )
}

function GroupCard({ group: g }: { group: ProductGroup }) {
  const [expanded, setExpanded] = useState(false)
  const range = g.max_price - g.min_price
  const bestItem = g.items.find(i => i.product_id === g.best_price_id)

  return (
    <div style={{
      background: 'var(--surface-2)', border: '1px solid var(--border)',
      borderRadius: 'var(--radius-lg)', overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{ padding: '16px 18px' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, flexWrap: 'wrap' }}>
          <div style={{ flex: 1, minWidth: 200 }}>
            <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 4 }}>{g.canonical_name}</h3>
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
              {g.items.length} penawaran · {g.category_path}
            </p>
            {g.important_specs?.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 5, marginTop: 8 }}>
                {g.important_specs.slice(0, 4).map(s => (
                  <span key={s} style={{
                    background: 'var(--surface-3)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-sm)', fontSize: 11, padding: '2px 8px',
                    color: 'var(--text-muted)',
                  }}>{s}</span>
                ))}
              </div>
            )}
          </div>

          {/* Price range */}
          <div style={{ textAlign: 'right', flexShrink: 0 }}>
            <p style={{ fontSize: 18, fontWeight: 700 }}>{formatRupiahShort(g.min_price)}</p>
            <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>– {formatRupiahShort(g.max_price)}</p>
            <p style={{ fontSize: 11, color: 'var(--text-faint)', marginTop: 2 }}>
              rata-rata {formatRupiahShort(g.avg_price)}
            </p>
          </div>
        </div>

        {/* Price bar */}
        <div style={{ marginTop: 14, marginBottom: 12 }}>
          <div style={{ height: 4, background: 'var(--surface-3)', borderRadius: 99, position: 'relative' }}>
            {g.items.map(item => {
              const pct = range > 0 ? ((item.price - g.min_price) / range) * 90 : 50
              return (
                <div key={item.product_id} title={`${item.shop_name}: ${formatRupiah(item.price)}`} style={{
                  position: 'absolute', left: `${pct}%`,
                  width: 10, height: 10, borderRadius: '50%', top: '50%', transform: 'translate(-50%, -50%)',
                  background: item.product_id === g.best_price_id ? 'var(--accent)' : 'var(--text-faint)',
                  border: item.product_id === g.best_price_id ? '2px solid rgba(34,197,94,0.4)' : 'none',
                  cursor: 'pointer',
                }} />
              )
            })}
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 5, fontSize: 11, color: 'var(--text-faint)' }}>
            <span>Termurah</span>
            <span>Termahal</span>
          </div>
        </div>

        {/* Best deal */}
        {bestItem && (
          <div style={{
            background: 'var(--accent-dim)', border: '1px solid rgba(34,197,94,0.15)',
            borderRadius: 'var(--radius-md)', padding: '10px 14px',
            display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap',
          }}>
            <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--accent)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>Harga Terbaik</span>
            <span className={`marketplace-badge ${marketplaceClass(bestItem.marketplace)}`}>{marketplaceLabel(bestItem.marketplace)}</span>
            <span style={{ fontWeight: 700 }}>{formatRupiah(bestItem.price)}</span>
            <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>{bestItem.shop_name}</span>
            {bestItem.is_official_store && <span style={{ background: 'rgba(34,197,94,0.1)', color: 'var(--accent)', fontSize: 10, fontWeight: 600, padding: '1px 7px', borderRadius: 99 }}>✓ Resmi</span>}
            {bestItem.rating > 0 && <span style={{ color: '#facc15', fontSize: 12 }}>★ {bestItem.rating.toFixed(1)}</span>}
            <a href={bestItem.url} target="_blank" rel="noopener noreferrer" style={{ marginLeft: 'auto', color: 'var(--accent)', fontSize: 12, fontWeight: 500 }}>
              Lihat →
            </a>
          </div>
        )}
      </div>

      {/* Toggle expand */}
      <button
        onClick={() => setExpanded(v => !v)}
        style={{
          width: '100%', background: 'none', border: 'none', cursor: 'pointer',
          borderTop: '1px solid var(--border)', padding: '10px 18px',
          color: 'var(--text-muted)', fontSize: 12, fontWeight: 500,
          display: 'flex', alignItems: 'center', gap: 6, transition: 'background 0.15s',
        }}
        onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-hover)')}
        onMouseLeave={e => (e.currentTarget.style.background = 'none')}
      >
        <span style={{ transition: 'transform 0.2s', transform: expanded ? 'rotate(180deg)' : 'none', display: 'inline-block' }}>▾</span>
        {expanded ? 'Sembunyikan' : `Lihat semua ${g.items.length} penawaran`}
      </button>

      {/* Expanded items */}
      {expanded && (
        <div style={{ borderTop: '1px solid var(--border)' }}>
          {g.items.map((item, idx) => (
            <div key={item.product_id} style={{
              display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap',
              padding: '12px 18px',
              borderBottom: idx < g.items.length - 1 ? '1px solid var(--border)' : 'none',
              background: item.product_id === g.best_price_id ? 'rgba(34,197,94,0.04)' : 'transparent',
            }}>
              <span className={`marketplace-badge ${marketplaceClass(item.marketplace)}`}>{marketplaceLabel(item.marketplace)}</span>
              <div style={{ flex: 1, minWidth: 140 }}>
                <p style={{ fontSize: 13, fontWeight: 500, marginBottom: 2 }}>{item.shop_name}</p>
                <p style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                  {item.shop_city} · ★ {item.rating.toFixed(1)} · {formatCount(item.count_review)} ulasan
                  {item.sold > 0 && ` · Terjual ${formatCount(item.sold)}`}
                </p>
              </div>
              <div style={{ textAlign: 'right' }}>
                <p style={{ fontSize: 15, fontWeight: 700 }}>{formatRupiah(item.price)}</p>
                {item.discount_percent > 0 && (
                  <p style={{ fontSize: 11, color: 'var(--text-muted)', textDecoration: 'line-through' }}>{formatRupiah(item.original_price)}</p>
                )}
              </div>
              {item.is_official_store && <span style={{ background: 'rgba(34,197,94,0.1)', color: 'var(--accent)', fontSize: 10, fontWeight: 600, padding: '2px 8px', borderRadius: 99 }}>✓ Resmi</span>}
              <a href={item.url} target="_blank" rel="noopener noreferrer" style={{ color: 'var(--text-muted)', fontSize: 12 }}>Buka →</a>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
```

---

## Step 14.7 — AI Insights Panel

**File:** `components/run-detail/AIInsightsPanel.tsx`

```tsx
'use client'
import { AISummaryResult, Product } from '@/lib/types'
import { formatRupiah, marketplaceClass, marketplaceLabel } from '@/lib/format'

const BADGES = ['Pilihan Terbaik', 'Termurah', 'Paling Terpercaya', 'Best Value', 'Top Rating']

interface Props {
  summary: AISummaryResult | null
  status: 'idle' | 'loading' | 'done' | 'error'
  products: Product[]
}

export function AIInsightsPanel({ summary, status, products }: Props) {
  if (status === 'loading' || status === 'idle') return (
    <div style={{ padding: '40px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 24 }}>
        <span style={{ fontSize: 24 }}>🤖</span>
        <div>
          <p style={{ fontWeight: 600, fontSize: 15 }}>Menganalisis produk...</p>
          <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>
            AI sedang mempelajari {products.length} produk untuk menemukan pilihan terbaik.
          </p>
        </div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 12 }}>
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="skeleton-box" style={{ height: 160, borderRadius: 'var(--radius-lg)' }} />
        ))}
      </div>
    </div>
  )

  if (status === 'error') return (
    <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
      <p style={{ fontSize: 18, marginBottom: 8 }}>⚠</p>
      <p>Analisis AI tidak tersedia saat ini.</p>
      <p style={{ fontSize: 13, marginTop: 6 }}>Kamu tetap bisa melihat semua produk di tab "Semua Produk".</p>
    </div>
  )

  if (!summary) return null

  const productMap = Object.fromEntries(products.map(p => [p.id, p]))

  return (
    <div>
      {/* Summary text */}
      <div style={{
        background: 'var(--surface-2)', border: '1px solid rgba(34,197,94,0.15)',
        borderRadius: 'var(--radius-lg)', padding: '18px 20px', marginBottom: 22,
        borderLeft: '3px solid var(--accent)',
      }}>
        <p style={{ fontSize: 12, fontWeight: 600, color: 'var(--accent)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 8 }}>
          🤖 Ringkasan AI
        </p>
        <p style={{ fontSize: 14, lineHeight: 1.7, color: 'var(--text)' }}>
          {summary.summary_text}
        </p>
      </div>

      {/* Recommendation cards */}
      {summary.recommended_items?.length > 0 && (
        <>
          <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-muted)', marginBottom: 14, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Rekomendasi Terbaik
          </p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 14 }}>
            {summary.recommended_items.map((rec, idx) => {
              const product = productMap[rec.product_id]
              return (
                <div key={rec.product_id} style={{
                  background: 'var(--surface-2)', border: '1px solid var(--border)',
                  borderRadius: 'var(--radius-lg)', padding: '16px',
                  display: 'flex', flexDirection: 'column', gap: 10,
                  transition: 'border-color 0.2s',
                }}>
                  {/* Badge */}
                  <span style={{
                    alignSelf: 'flex-start',
                    background: idx === 0 ? 'var(--accent-dim)' : 'var(--surface-3)',
                    color: idx === 0 ? 'var(--accent)' : 'var(--text-muted)',
                    border: `1px solid ${idx === 0 ? 'rgba(34,197,94,0.25)' : 'var(--border)'}`,
                    fontSize: 10, fontWeight: 700, padding: '3px 10px', borderRadius: 99,
                    textTransform: 'uppercase', letterSpacing: '0.05em',
                  }}>
                    {BADGES[idx] ?? `#${idx + 1}`}
                  </span>

                  {product ? (
                    <>
                      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
                        {product.image_url && (
                          <img src={product.image_url} alt={product.name}
                            style={{ width: 56, height: 56, objectFit: 'cover', borderRadius: 'var(--radius-sm)', flexShrink: 0 }}
                            onError={e => (e.currentTarget.style.display = 'none')}
                          />
                        )}
                        <div>
                          <p style={{ fontSize: 13, fontWeight: 600, lineHeight: 1.4, marginBottom: 4,
                            display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden',
                          }}>{product.name}</p>
                          <p style={{ fontSize: 16, fontWeight: 700 }}>{formatRupiah(product.price)}</p>
                        </div>
                      </div>

                      <div style={{ display: 'flex', alignItems: 'center', gap: 7, flexWrap: 'wrap' }}>
                        <span className={`marketplace-badge ${marketplaceClass(product.marketplace)}`}>
                          {marketplaceLabel(product.marketplace)}
                        </span>
                        {product.is_official_store && (
                          <span style={{ background: 'rgba(34,197,94,0.1)', color: 'var(--accent)', fontSize: 10, fontWeight: 600, padding: '1px 7px', borderRadius: 99 }}>✓ Resmi</span>
                        )}
                        {product.rating > 0 && (
                          <span style={{ fontSize: 12, color: '#facc15' }}>★ {product.rating.toFixed(1)}</span>
                        )}
                        <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{product.shop_name}</span>
                      </div>
                    </>
                  ) : (
                    <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>Produk: {rec.product_id}</p>
                  )}

                  {/* AI reason */}
                  <p style={{ fontSize: 12, color: 'var(--text-muted)', lineHeight: 1.5, borderTop: '1px solid var(--border)', paddingTop: 10 }}>
                    💬 {rec.reason}
                  </p>

                  {product && (
                    <a href={product.url} target="_blank" rel="noopener noreferrer" style={{
                      display: 'block', textAlign: 'center',
                      background: idx === 0 ? 'var(--accent)' : 'var(--surface-3)',
                      color: idx === 0 ? '#052e16' : 'var(--text)',
                      border: `1px solid ${idx === 0 ? 'transparent' : 'var(--border)'}`,
                      borderRadius: 'var(--radius-md)', padding: '8px', fontSize: 12, fontWeight: 600,
                      textDecoration: 'none',
                    }}>
                      Lihat di {marketplaceLabel(product.marketplace)} →
                    </a>
                  )}
                </div>
              )
            })}
          </div>
        </>
      )}
    </div>
  )
}
```

---

## Step 14.8 — Detail Teknis (Collapsible)

**File:** `components/run-detail/RunTechDetails.tsx`

```tsx
'use client'
import { Run } from '@/lib/types'
import { formatDate, formatDuration } from '@/lib/format'

export function RunTechDetails({ run }: { run: Run }) {
  const rows = [
    { label: 'Run ID',       value: run.id },
    { label: 'Status',       value: run.status },
    { label: 'Marketplace',  value: run.marketplace },
    { label: 'Dibuat',       value: formatDate(run.created_at) },
    { label: 'Mulai',        value: run.started_at  ? formatDate(run.started_at)  : '-' },
    { label: 'Selesai',      value: run.finished_at ? formatDate(run.finished_at) : '-' },
    { label: 'Durasi',       value: formatDuration(run.started_at, run.finished_at) },
    { label: 'Jumlah Item',  value: String(run.item_count) },
    { label: 'Error',        value: run.error_message ?? '-' },
    { label: 'Input',        value: JSON.stringify(run.input ?? {}, null, 2) },
  ]

  return (
    <div style={{
      background: 'var(--surface-2)', border: '1px solid var(--border)',
      borderRadius: 'var(--radius-lg)', overflow: 'hidden',
    }}>
      <div style={{ padding: '14px 18px', borderBottom: '1px solid var(--border)' }}>
        <p style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
          Detail Teknis
        </p>
      </div>
      {rows.map(r => (
        <div key={r.label} style={{
          display: 'grid', gridTemplateColumns: '140px 1fr',
          padding: '10px 18px', borderBottom: '1px solid var(--border)',
          fontSize: 13,
        }}>
          <span style={{ color: 'var(--text-muted)', fontWeight: 500 }}>{r.label}</span>
          <span style={{
            color: 'var(--text)', fontFamily: r.label === 'Run ID' || r.label === 'Input' ? 'monospace' : 'inherit',
            fontSize: r.label === 'Input' ? 12 : 13,
            whiteSpace: r.label === 'Input' ? 'pre-wrap' : 'normal',
            wordBreak: 'break-all',
          }}>{r.value}</span>
        </div>
      ))}
    </div>
  )
}
```

---

## Step 14.9 — Halaman Riwayat

**File:** `app/riwayat/page.tsx`

```tsx
import { api } from '@/lib/api'
import { RiwayatClient } from '@/components/riwayat/RiwayatClient'

export default async function RiwayatPage() {
  const data = await api.getRuns(100, 0).catch(() => ({ runs: [], total: 0 }))
  return <RiwayatClient initialRuns={data.runs} />
}
```

**File:** `components/riwayat/RiwayatClient.tsx`

```tsx
'use client'
import { useState, useEffect } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Run } from '@/lib/types'
import { api } from '@/lib/api'
import { formatDate, formatDuration, humanStatus, marketplaceLabel } from '@/lib/format'
import { StatusBadge } from '@/components/ui/StatusBadge'

export function RiwayatClient({ initialRuns }: { initialRuns: Run[] }) {
  const [runs, setRuns] = useState<Run[]>(initialRuns)
  const router = useRouter()

  // Auto-refresh jika ada run yang masih aktif
  useEffect(() => {
    const hasActive = runs.some(r => r.status === 'RUNNING' || r.status === 'QUEUED')
    if (!hasActive) return
    const iv = setInterval(async () => {
      const data = await api.getRuns(100, 0).catch(() => null)
      if (data) setRuns(data.runs)
    }, 3000)
    return () => clearInterval(iv)
  }, [runs])

  const keyword = (run: Run) => (run.input as Record<string, string>)?.keyword ?? 'Pencarian'

  return (
    <div style={{ maxWidth: 800, margin: '0 auto', padding: '32px 20px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 28 }}>
        <div>
          <h1 style={{ fontSize: 20, fontWeight: 700 }}>Riwayat Pencarian</h1>
          <p style={{ color: 'var(--text-muted)', fontSize: 13, marginTop: 4 }}>{runs.length} pencarian tersimpan</p>
        </div>
        <Link href="/" style={{
          background: 'var(--accent)', color: '#052e16',
          borderRadius: 'var(--radius-md)', padding: '9px 18px', fontSize: 13, fontWeight: 600, textDecoration: 'none',
        }}>
          + Pencarian Baru
        </Link>
      </div>

      {runs.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 20px', color: 'var(--text-muted)' }}>
          <p style={{ fontSize: 32, marginBottom: 16 }}>◎</p>
          <h3 style={{ fontSize: 17, fontWeight: 600, color: 'var(--text)', marginBottom: 8 }}>Belum ada pencarian</h3>
          <p style={{ marginBottom: 20 }}>Mulai cari produk dan bandingkan harga dari berbagai toko.</p>
          <Link href="/" style={{ color: 'var(--accent)', fontWeight: 600 }}>Mulai Cari →</Link>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {runs.map(run => (
            <div key={run.id} style={{
              background: 'var(--surface-2)', border: '1px solid var(--border)',
              borderRadius: 'var(--radius-lg)', padding: '16px 18px',
              transition: 'border-color 0.15s',
            }}
            onMouseEnter={e => (e.currentTarget.style.borderColor = 'var(--border-hover)')}
            onMouseLeave={e => (e.currentTarget.style.borderColor = 'var(--border)')}>

              <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' }}>
                <div style={{ flex: 1 }}>
                  <p style={{ fontSize: 15, fontWeight: 600, marginBottom: 6 }}>"{keyword(run)}"</p>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', color: 'var(--text-muted)', fontSize: 12 }}>
                    <StatusBadge status={run.status} />
                    <span>{marketplaceLabel(run.marketplace)}</span>
                    {run.item_count > 0 && <span>{run.item_count} produk</span>}
                    <span>{formatDate(run.created_at)}</span>
                    {run.started_at && run.finished_at && (
                      <span>{formatDuration(run.started_at, run.finished_at)}</span>
                    )}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
                  <Link href={`/pencarian/${run.id}`} style={{
                    background: 'var(--surface-3)', border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-md)', padding: '7px 14px',
                    color: 'var(--text)', fontSize: 12, fontWeight: 500, textDecoration: 'none',
                  }}>
                    Lihat Hasil
                  </Link>
                  <button
                    onClick={() => router.push('/')}
                    style={{
                      background: 'none', border: '1px solid var(--border)',
                      borderRadius: 'var(--radius-md)', padding: '7px 14px',
                      color: 'var(--text-muted)', fontSize: 12, cursor: 'pointer',
                    }}
                  >
                    Ulangi
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
```

---

## Step 14.10 — Shared UI Components

**File:** `components/ui/StatusBadge.tsx`

```tsx
import { humanStatus } from '@/lib/format'

export function StatusBadge({ status }: { status: string }) {
  const cls = `status-badge status-${status.toLowerCase()}`
  return (
    <span className={cls}>
      <span className="dot" />
      {humanStatus(status)}
    </span>
  )
}
```

**File:** `components/ui/ProductCardSkeleton.tsx`

```tsx
export function ProductCardSkeleton() {
  return (
    <div style={{
      background: 'var(--surface-2)', border: '1px solid var(--border)',
      borderRadius: 'var(--radius-lg)', overflow: 'hidden',
    }}>
      <div className="skeleton-box" style={{ aspectRatio: '1' }} />
      <div style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div className="skeleton-box" style={{ height: 14, width: '80%' }} />
        <div className="skeleton-box" style={{ height: 12, width: '50%' }} />
        <div className="skeleton-box" style={{ height: 18, width: '65%', marginTop: 4 }} />
        <div className="skeleton-box" style={{ height: 32, marginTop: 8, borderRadius: 8 }} />
      </div>
    </div>
  )
}
```

---

## Step 14.11 — Verification Checklist

1. `npm run build` — tidak ada TypeScript error
2. Buka `http://localhost:3000`:
   - Search hub tampil, filter tersembunyi, recent searches dari localStorage
3. Submit pencarian:
   - Redirect ke `/pencarian/[id]`
   - Banner "Sedang mencari..." muncul dengan animasi
   - Counter produk update realtime saat run masih berjalan
4. Setelah selesai:
   - Tab "Semua Produk": grid tampil dengan marketplace badge berwarna
   - Kartu produk: harga, diskon, rating, toko, kota, tombol ke marketplace
   - Filter (harga, toko resmi, sort) berfungsi tanpa reload
   - Tab "Kelompok Serupa": group cards dengan bar harga dan expand items
   - Tab "🤖 Rekomendasi AI": summary text + recommendation cards dengan alasan
   - Tab "Detail Teknis": semua metadata run tersedia
5. Buka `/riwayat`:
   - List pencarian dengan kartu, status manusiawi, auto-refresh bila ada run aktif
6. Mobile (375px):
   - Bottom navigation muncul, sidebar hilang
   - Grid produk 2 kolom, filter accessible
   - Cards tidak overflow
7. Cek konten:
   - Tidak ada teks "SUCCEEDED", "normalized", "canonical_key", "run_id" yang terlihat user
   - Semua harga format Rupiah Indonesia
   - Semua tanggal format Indonesia

Update AGENTS.md: Phase 14 ✅
