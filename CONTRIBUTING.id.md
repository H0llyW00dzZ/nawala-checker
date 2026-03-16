# Berkontribusi pada Nawala Checker

[![Read in English](https://img.shields.io/badge/🇬🇧-Read%20in%20English-blue)](CONTRIBUTING.md)

Terima kasih atas minat Anda untuk berkontribusi pada SDK **Nawala Checker**! Repositori ini mematuhi standar pengembangan Go SDK yang ketat. Kami menyambut baik kontribusi yang meningkatkan keandalan, performa, atau dokumentasi.

Harap baca [Kode Etik (Code of Conduct)](CODE_OF_CONDUCT.id.md) kami sebelum berpartisipasi dalam komunitas kami.

## Struktur Proyek

Proyek ini mengikuti tata letak (layout) Go SDK standar untuk memastikan penggunaan idiomatik dan meminimalisir ketergantungan (dependency overhead).

```text
nawala-checker/
├── cmd/
│   └── nawala/       # Titik masuk (entry point) CLI (main.go).
├── internal/
│   └── cli/          # Perintah CLI, pemuatan konfigurasi, format output, dan teks penggunaan.
├── src/
│   └── nawala/       # Logika pengecekan DNS, opsi-opsi, tipe structs, dan cache.
├── examples/         # Contoh kode yang dapat dijalankan (basic, custom, status, hotreload, streaming, pooling).
├── .github/          # Alur kerja (workflows) CI/CD dan template GitHub.
├── Makefile          # Perintah untuk build, test, dan coverage.
├── README.md         # Dokumentasi utama (Bahasa Inggris).
└── README.id.md      # Dokumentasi terjemahan (Bahasa Indonesia).
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
*   Pastikan semua opsi konfigurasi baru menggunakan pola **Functional Options** di [`src/nawala/option.go`](src/nawala/option.go).
*   Semua metode yang melakukan I/O (Input/Output) harus menerima `context.Context` sebagai argumen pertama.
*   Hindari menambahkan dependensi pihak ketiga kecuali benar-benar diperlukan.

**Pengujian (Testing)**:
*   Kami mewajibkan tes untuk semua jalur kode (code paths) baru.
*   Periksa cakupan pengujian (test coverage) Anda secara lokal sebelum mengirim:
    ```bash
    make test-cover
    ```

**Dokumentasi (Sinkronisasi Multibahasa & Kode)**:
*   `nawala-checker` mengelola dokumentasi dalam bahasa Inggris ([`README.md`](README.md)) dan bahasa Indonesia ([`README.id.md`](README.id.md)), serta dokumentasi level-paket (GoDoc) di [`src/nawala/docs.go`](src/nawala/docs.go) dan [`internal/cli/docs.go`](internal/cli/docs.go).
*   Jika Pull Request Anda menambahkan fitur baru, mengubah API publik, atau memodifikasi perilaku yang ada, Anda **wajib menyinkronkan semua sumber dokumentasi** untuk memastikan keakuratan teknis dan konsistensi di seluruh sumber dokumentasi.

### Menyinkronkan Dokumentasi

> [!NOTE]
> Proses ini hanya berlaku ketika pengguna manusia menggunakan opencode.ai. Jika tidak, ikuti langkah-langkah manual di bawah ini.

Ketika membuat perubahan yang memengaruhi dokumentasi, jalankan perintah `sync-docs` melalui opencode.ai untuk memastikan konsistensi di seluruh dokumentasi multibahasa, GoDoc, contoh, dan teks penggunaan CLI.

#### Sumber Dokumentasi yang Akan Disinkronkan

1. **File Dokumentasi Utama**
   - `README.md` - Dokumentasi bahasa Inggris (utama)
   - `README.id.md` - Dokumentasi bahasa Indonesia (terjemahan)

2. **GoDoc Level-Paket**
   - `src/nawala/docs.go` - Dokumentasi paket SDK inti
   - `internal/cli/docs.go` - Dokumentasi paket CLI

3. **Contoh Kode**
   - direktori `examples/` - Contoh kode yang dapat dijalankan yang mendemonstrasikan penggunaan

4. **Teks Penggunaan CLI**
   - direktori `internal/cli/usage/` - Teks bantuan CLI yang di-embed

5. **Panduan Berkontribusi**
   - `CONTRIBUTING.md` - Panduan berkontribusi bahasa Inggris
   - `CONTRIBUTING.id.md` - Panduan berkontribusi bahasa Indonesia (terjemahan)

#### Proses Sinkronisasi

1. **Identifikasi Perubahan**: Tinjau perubahan kode Anda untuk menentukan dokumentasi apa yang perlu diperbarui (opsi konfigurasi baru, tanda tangan fungsi yang berubah, perintah atau flag CLI baru, perilaku atau kondisi error yang dimodifikasi).

2. **Perbarui File README**: Perbarui kedua versi bahasa (`README.md` dan `README.id.md`).

3. **Perbarui GoDoc**: Pastikan dokumentasi paket mencerminkan API baru/yang berubah di `src/nawala/docs.go` dan `internal/cli/docs.go`.

4. **Perbarui Contoh**: Tambah atau modifikasi contoh di direktori `examples/` untuk mendemonstrasikan fitur baru.

5. **Perbarui Penggunaan CLI**: Untuk perubahan CLI, perbarui teks penggunaan yang di-embed di `internal/cli/usage/`.

6. **Verifikasi Konsistensi**: Jalankan pengujian untuk memastikan semua sumber dokumentasi akurat secara teknis:
   ```bash
   make test-verbose
   ```

#### Persyaratan Multibahasa

Karena proyek ini mempertahankan dokumentasi dalam bahasa Inggris dan Indonesia:
- Jaga keakuratan teknis konsisten di antara bahasa
- Perbarui kedua file README secara simultan
- Pertahankan struktur dan contoh yang sama di kedua bahasa

#### Pesan Commit

Gunakan format commit konvensional:
```
docs: sinkronkan dokumentasi untuk [deskripsi]

- [+] Perbarui README.md dengan perubahan
- [+] Perbarui CONTRIBUTING.md dan CONTRIBUTING.id.md
- [+] Perbarui GoDoc di docs.go
- [+] Tambah/modifikasi contoh jika diperlukan
- [+] Perbarui teks penggunaan CLI
```

#### Daftar Periksa Verifikasi

- [ ] README.md diperbarui dengan fitur/perubahan baru
- [ ] README.id.md diperbarui (bahasa Indonesia)
- [ ] CONTRIBUTING.md diperbarui
- [ ] CONTRIBUTING.id.md diperbarui (bahasa Indonesia)
- [ ] GoDoc di file docs.go diperbarui
- [ ] Direktori examples diperbarui jika diperlukan
- [ ] Teks penggunaan CLI diperbarui untuk perubahan CLI

### 3. Melakukan Commit dan Pemformatan
Sebelum melakukan komit, pastikan kode Anda diformat dengan benar:
```bash
gofmt -s -w .
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
