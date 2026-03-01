# Berkontribusi pada Nawala Checker

[![Read in English](https://img.shields.io/badge/🇬🇧-Read%20in%20English-blue)](CONTRIBUTING.md)

Terima kasih atas minat Anda untuk berkontribusi pada SDK **Nawala Checker**! Repositori ini mematuhi standar pengembangan Go SDK yang ketat. Kami menyambut baik kontribusi yang meningkatkan keandalan, performa, atau dokumentasi.

Harap baca [Kode Etik (Code of Conduct)](CODE_OF_CONDUCT.id.md) kami sebelum berpartisipasi dalam komunitas kami.

## Struktur Proyek

Proyek ini mengikuti tata letak (layout) Go SDK standar untuk memastikan penggunaan idiomatik dan meminimalisir ketergantungan (dependency overhead).

```text
nawala-checker/
├── src/
│   └── nawala/      # Logika pengecekan DNS, opsi-opsi, tipe structs, dan cache.
├── examples/        # Contoh kode yang dapat dijalankan (basic, custom, status, hotreload).
├── .github/         # Alur kerja (workflows) CI/CD dan template GitHub.
├── Makefile         # Perintah untuk build, test, dan coverage.
├── README.md        # Dokumentasi utama (Bahasa Inggris).
└── README.id.md     # Dokumentasi terjemahan (Bahasa Indonesia).
```

## Persiapan dan Verifikasi

Untuk memastikan proses kontribusi yang bersih, silakan ikuti Alur Kerja Fork-First:

1. **Fork repositori** ke akun GitHub Anda sendiri.
2. **Kloning (Clone) fork Anda**:
   ```bash
   git clone https://github.com/USERNAME_ANDA/nawala-checker.git
   cd nawala-checker
   ```
3. **Verifikasi penyiapan Anda** dengan menjalankan rangkaian pengujian:
   ```bash
   make test-verbose
   ```
   *Catatan: Jika Anda tidak berada di jaringan Indonesia, beberapa pengujian DNS langsung (live) mungkin gagal atau dilewati. Anda dapat menjalankan pengujian unit (unit tests) saja menggunakan `make test-short`.*

## Siklus Hidup Kontribusi (Contribution Lifecycle)

### 1. Percabangan (Branching)
Buat branch (cabang) khusus untuk pekerjaan Anda. Gunakan awalan (prefix) yang deskriptif:
*   `feature/` untuk kemampuan/fitur baru (contoh: `feature/redis-cache`)
*   `fix/` untuk perbaikan bug (contoh: `fix/edns0-parsing`)
*   `docs/` untuk pembaruan dokumentasi
*   `chore/` untuk pemeliharaan (contoh: pembaruan CI/CD)

```bash
git checkout -b feature/nama-fitur-anda
```

### 2. Melakukan Perubahan

**Standar Kode**:
*   Pastikan semua opsi konfigurasi baru menggunakan pola **Functional Options** di `src/nawala/option.go`.
*   Semua metode yang melakukan I/O (Input/Output) harus menerima `context.Context` sebagai argumen pertama.
*   Hindari menambahkan dependensi pihak ketiga kecuali benar-benar diperlukan.

**Pengujian (Testing)**:
*   Kami mewajibkan tes untuk semua jalur kode (code paths) baru.
*   Periksa cakupan pengujian (test coverage) Anda secara lokal sebelum mengirim:
    ```bash
    make test-cover
    ```

**Dokumentasi (Sinkronisasi Multibahasa & Kode)**:
*   `nawala-checker` mengelola dokumentasi dalam bahasa Inggris (`README.md`) dan bahasa Indonesia (`README.id.md`), serta dokumentasi level-paket (GoDoc) di `src/nawala/docs.go`.
*   Jika Pull Request Anda menambahkan fitur baru, mengubah API publik, atau memodifikasi perilaku yang ada, Anda **wajib memperbarui `README.md`, `README.id.md`, `src/nawala/docs.go`, serta kode terkait di direktori `examples/`** untuk memastikan keakuratan teknis dan konsistensi di seluruh sumber dokumentasi.

### 3. Melakukan Commit dan Pemformatan
Sebelum melakukan komit, pastikan kode Anda diformat dengan benar:
```bash
gofmt -s -w ./src/...
```

Tulis pesan komit (commit messages) yang jelas dan deskriptif. Kami mendorong penggunaan Conventional Commits:
```
feat: add custom EDNS0 size configuration
fix: resolve race condition in cache expiration
docs: update hot-reload example in README
```

### 4. Membuka Pull Request
1. Push branch Anda ke repository fork Anda.
2. Buka Pull Request yang ditujukan ke branch `master` pada repositori upstream (repositori utama).
3. Isi template PR, jelaskan apa yang Anda ubah dan alasannya.
4. Pipeline CI akan secara otomatis melakukan linting, memformat, dan menjalankan pengujian (test suite) di berbagai versi Go dengan race detector diaktifkan.

## Tinjauan Kode (Code Review)
Maintainer akan meninjau (me-review) PR Anda. Kami mungkin meminta perubahan agar selaras dengan arsitektur inti yang dijelaskan dalam standar kami (Functional Options, Context-First, Typed Errors). Setelah disetujui dan semua pemeriksaan CI berhasil, PR Anda akan di-merge!

---
*Dengan berkontribusi pada repositori ini, Anda setuju bahwa kontribusi Anda akan dilisensikan di bawah [Lisensi BSD 3-Clause](LICENSE) dari proyek ini.*
