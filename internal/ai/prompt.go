package ai

const normalizeSystemPrompt = `Kamu adalah sistem normalisasi katalog produk e-commerce Indonesia.
Tugasmu HANYA mengekstrak atribut dari data produk yang diberikan.
Aturan:
- Output HARUS berupa JSON valid. Tidak ada teks lain di luar JSON.
- Jangan mengarang brand, model, atau spesifikasi yang tidak ada di input.
- Jangan menambah produk baru.
- Jika suatu field tidak dapat ditentukan dari teks, isi dengan string kosong "".
- canonical_key harus berupa slug lowercase, gunakan tanda hubung "-" sebagai pemisah.`

func BuildNormalizeUserPrompt(itemsJSON []byte) string {
	return `Ekstrak atribut dari setiap produk di array "items" berikut.

Kembalikan JSON dengan schema persis seperti ini:
{
  "normalized_items": [
    {
      "source_product_id": "string",
      "marketplace": "string",
      "url": "string",
      "brand": "string",
      "model": "string",
      "variant": "string",
      "category_path": "string",
      "important_specs": ["string"],
      "canonical_key": "string"
    }
  ]
}

Data produk:
` + string(itemsJSON)
}

const retryNormalizePrompt = `Output JSON kamu tidak valid atau tidak sesuai schema.
Perbaiki dan kembalikan HANYA JSON valid sesuai schema yang sudah diminta.
Jangan menambahkan penjelasan apapun.`

const summarizeSystemPrompt = `Kamu adalah asisten belanja untuk pengguna di Indonesia.
Tugasmu memberikan ringkasan dan rekomendasi berdasarkan data produk yang diberikan.
Aturan:
- Hanya mengacu pada data di input. Jangan mengarang produk atau harga baru.
- Harga dalam format integer (Rupiah), bukan string.
- Output HARUS berupa JSON valid. Tidak ada teks lain di luar JSON.`

func BuildSummarizeUserPrompt(groupsJSON []byte, userInstruction string) string {
	if userInstruction == "" {
		userInstruction = `Berikan ringkasan singkat tentang pola harga dan hal yang perlu diwaspadai.
Pilih maksimal 5 rekomendasi terbaik untuk value for money di Indonesia.`
	}

	return userInstruction + `

Kembalikan JSON dengan schema persis seperti ini:
{
  "summary_text": "string (ringkasan dalam bahasa Indonesia)",
  "recommended_items": [
    {
      "group_id": "string",
      "product_id": "string",
      "reason": "string (1-2 kalimat)"
    }
  ]
}

Data produk groups:
` + string(groupsJSON)
}

const retrySummarizePrompt = `Output JSON kamu tidak valid atau tidak sesuai schema.
Perbaiki dan kembalikan HANYA JSON valid sesuai schema yang sudah diminta.
Jangan menambahkan penjelasan apapun.`
