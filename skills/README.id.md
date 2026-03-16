# 🤖 Nawala Checker — Skill AI

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25.6-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Read in English](https://img.shields.io/badge/🇬🇧-Read%20in%20English-blue)](README.md)

Direktori ini berisi **definisi skill** yang memungkinkan bot dan agen AI untuk berintegrasi dengan nawala-checker. Setiap skill menyediakan instruksi terstruktur untuk menjalankan nawala-checker melalui **CLI** (semua bahasa pemrograman) atau **Go SDK** (khusus Go).

## Apa Itu Skill?

Skill adalah file instruksi standar (`SKILL.md`) yang memberitahu agen AI cara menggunakan sebuah tool. Ketika bot AI diarahkan ke direktori ini, bot dapat menemukan dan menjalankan kemampuan nawala-checker secara otomatis — memeriksa domain, memantau kesehatan DNS, dan mengelola konfigurasi.

## Skill yang Tersedia

| Skill | Deskripsi |
|---|---|
| [`check_domains`](check_domains/SKILL.md) | Periksa apakah domain diblokir oleh filter DNS ISP Indonesia |
| [`dns_status`](dns_status/SKILL.md) | Periksa kesehatan dan latensi server DNS yang dikonfigurasi |
| [`config_inspect`](config_inspect/SKILL.md) | Periksa konfigurasi efektif dan buat file konfigurasi |

## Prasyarat

### CLI (Diperlukan untuk integrasi CLI)

```bash
go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
```

### Go SDK (Diperlukan untuk integrasi SDK)

```bash
go get github.com/H0llyW00dzZ/nawala-checker
```

> [!IMPORTANT]
> SDK ini membutuhkan **jaringan Indonesia** agar berfungsi dengan benar. Server DNS Nawala hanya mengembalikan respons pemblokiran ketika di-query dari dalam Indonesia.

## Cara Pengaturan

### 1. Clone Repositori

Clone repositori nawala-checker untuk mendapatkan direktori skill:

```bash
git clone https://github.com/H0llyW00dzZ/nawala-checker.git
```

### 2. Arahkan Agen AI ke Direktori Skills

Konfigurasikan framework AI Anda untuk membaca dari direktori `skills/` yang sudah di-clone:

- **[openclaw](https://openclaw.ai)** — Tambahkan path direktori skill ke konfigurasi agen:
  ```
  nawala-checker/skills/
  ```
- **[opencode](https://opencode.ai)** — Salin atau buat symlink direktori `skills/` ke proyek Anda (otomatis ditemukan)
- **[crush](https://github.com/charmbracelet/crush)** — Referensikan setiap `SKILL.md` sebagai definisi tool
- **Agen umum** — Arahkan agen ke direktori `skills/` atau file `SKILL.md` individual

### 3. Verifikasi CLI Tersedia

Pastikan binary `nawala` ada di `PATH` Anda:

```bash
nawala --version
```

Jika perintah tidak ditemukan, instal terlebih dahulu:

```bash
go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
```

### 4. Mulai Menggunakan Skill

Minta agen AI Anda untuk memeriksa domain:

```
> Periksa apakah reddit.com dan google.com diblokir oleh DNS Indonesia
```

Agen akan menemukan skill `check_domains` dan menjalankan:

```bash
nawala check --format json reddit.com google.com
```

## Metode Integrasi

Setiap skill mendukung dua pendekatan integrasi:

| Metode | Bahasa | Cocok Untuk |
|---|---|---|
| **CLI** | Semua | Python, TypeScript, atau framework AI non-Go |
| **Go SDK** | Go | Tool AI berbasis Go (opencode, openclaw, crush, dll.) |

### Contoh CLI

```bash
# Agen AI menjalankan perintah ini dan mem-parse output JSON
nawala check --format json google.com reddit.com
```

```json
{
  "nawala": {
    "version": "0.7.1",
    "result": [
      {"domain": "google.com", "blocked": false, "server": "180.131.144.144"},
      {"domain": "reddit.com", "blocked": true, "server": "180.131.144.144"}
    ]
  }
}
```

### Contoh Go SDK

```go
checker := nawala.New()
defer checker.Close()

results, _ := checker.Check(ctx, "google.com", "reddit.com")
for _, r := range results {
    fmt.Printf("%s: blocked=%v\n", r.Domain, r.Blocked)
}
```

## Struktur Direktori

```
skills/
├── README.md                   # Versi Bahasa Inggris
├── README.id.md                # File ini (Bahasa Indonesia)
├── check_domains/
│   └── SKILL.md                # Skill pemeriksaan pemblokiran domain
├── dns_status/
│   └── SKILL.md                # Skill kesehatan server DNS
└── config_inspect/
    └── SKILL.md                # Skill inspeksi konfigurasi
```
