# PHASE 9 — Final Verification & Checklist

## Objective
Verifikasi end-to-end bahwa seluruh sistem berjalan dengan benar sebelum dianggap selesai.

---

## Full Checklist

### Build & Test
- [ ] `go build ./...` sukses tanpa warning
- [ ] `go test ./... -v` semua test PASS
- [ ] `make build` menghasilkan binary `bin/api` dan `bin/worker`

### Infrastructure
- [ ] `make dev` → semua 4 container (postgres, redis, api, worker) status healthy
- [ ] `docker compose ps` menunjukkan semua service Up
- [ ] `make migrate` → tabel `runs` terbuat di PostgreSQL
- [ ] `curl http://localhost:8080/health` → `{"status":"ok"}`

### Core Flow
```bash
# 1. Submit job
RUN_ID=$(curl -s -X POST http://localhost:8080/v1/scrape/tokopedia/search \
  -H "Content-Type: application/json" \
  -d '{"keyword":"iphone 15","max_items":10,"sort_by":"price_asc"}' | jq -r .run_id)

echo "Run ID: $RUN_ID"

# 2. Cek status (tunggu beberapa detik)
curl -s http://localhost:8080/v1/runs/$RUN_ID | jq .status

# 3. Setelah SUCCEEDED, cek hasil
curl -s http://localhost:8080/v1/runs/$RUN_ID | jq '.result | length'
curl -s http://localhost:8080/v1/runs/$RUN_ID | jq '.result[0]'

# 4. List semua runs
curl -s "http://localhost:8080/v1/runs?limit=5" | jq .

# 5. Delete run
curl -s -X DELETE http://localhost:8080/v1/runs/$RUN_ID
```

### Expected Results
- [ ] Status berubah: `QUEUED` → `RUNNING` → `SUCCEEDED`
- [ ] `result` berisi array produk (minimal 1 item untuk keyword umum)
- [ ] Setiap produk punya field: `id`, `name`, `price`, `url`, `shop_name`, `marketplace: "tokopedia"`
- [ ] Produk diurutkan harga termurah dulu (karena `sort_by: price_asc`)
- [ ] Worker log menunjukkan: `"scrape completed" item_count=N`
- [ ] `DELETE /v1/runs/:id` return 204 dan run terhapus dari DB

### Edge Cases
- [ ] Submit dengan `keyword: ""` → response 400 Bad Request
- [ ] Submit dengan `max_items: 500` → otomatis di-cap ke 200
- [ ] Submit dengan `sort_by: "invalid"` → response 400
- [ ] `GET /v1/runs/nonexistent-id` → response 404
- [ ] `GET /v1/runs` → response 200 dengan pagination fields

---

## Jika Ada Yang Gagal

### Scraper return empty result
Kemungkinan GraphQL query string atau field mapping sudah berubah dari Tokopedia.
Buka browser → `tokopedia.com/search?q=laptop` → DevTools Network tab →
cari request ke `gql.tokopedia.com` → copy actual query dan response,
lalu update `scraper.go` dan `parser.go` sesuai.

### Worker tidak pick up job
Cek Redis connection: `docker compose exec redis redis-cli ping`
Cek worker logs: `make logs-worker`

### Database error
Cek migration sudah dijalankan: `make migrate`
Cek DATABASE_URL di .env sudah benar.

---

## Done! 🎉

Jika semua checklist hijau, Tokopedia Scraper Service sudah production-ready.

**Selanjutnya yang bisa ditambahkan:**
- Shopee scraper (implementasikan `MarketplaceScraper` interface yang sama)
- AI normalization layer (cross-marketplace product dedup)
- Simple web dashboard untuk monitor runs
- Scheduled jobs (wishlist price tracker)
