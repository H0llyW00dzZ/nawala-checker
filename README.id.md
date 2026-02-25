# Nawala Checker

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25.6-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Go Report Card](https://goreportcard.com/badge/github.com/H0llyW00dzZ/nawala-checker)](https://goreportcard.com/report/github.com/H0llyW00dzZ/nawala-checker)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/H0llyW00dzZ/nawala-checker/graph/badge.svg?token=3GU9QRYAUX)](https://codecov.io/gh/H0llyW00dzZ/nawala-checker)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/H0llyW00dzZ/nawala-checker)

SDK Go untuk memeriksa apakah domain diblokir oleh filter DNS ISP Indonesia (Nawala/Kominfo (sekarang Komdigi)). SDK ini bekerja dengan menanyakan server DNS yang dapat dikonfigurasi dan memindai respons untuk kata kunci pemblokiran seperti pengalihan `internetpositif.id` atau indikator EDE `trustpositif.komdigi.go.id`.

> [!IMPORTANT]
> SDK ini membutuhkan **jaringan Indonesia** agar berfungsi dengan benar. Server DNS Nawala hanya mengembalikan respons pemblokiran saat ditanyai dari dalam Indonesia. Pastikan koneksi Anda menggunakan IP Indonesia murni tanpa perutean melalui jaringan di luar Indonesia.

> [!NOTE]
> **SDK ini tidak usang atau ketinggalan zaman.** Meskipun ada rumor bahwa proyek Nawala yang asli mungkin berhenti beroperasi, modul ini tetap menjadi **perangkat pemeriksaan DNS tujuan umum** yang dibangun dari dasar dengan konfigurasi server dan klien DNS yang dapat disesuaikan. Anda dapat mengarahkannya ke server DNS mana pun, menentukan kata kunci pemblokiran Anda sendiri, dan memasang instance `*dns.Client` kustom (TCP, DNS-over-TLS, dialer kustom, dll.). Server Nawala default hanyalah default yang sudah dikonfigurasi sebelumnya; SDK itu sendiri sepenuhnya independen dan dipelihara secara aktif.

> [!TIP]
> Jika berjalan pada infrastruktur cloud (misalnya, VPS, microservice, k8s), lebih baik mengimplementasikan server DNS sendiri menggunakan jaringan Indonesia, kemudian dari infrastruktur cloud cukup memanggilnya.

## Fitur

- **Pemeriksaan domain serentak** — periksa beberapa domain secara paralel dengan satu panggilan
- **Failover server DNS** — fallback otomatis ke server sekunder ketika server utama gagal
- **Coba lagi dengan backoff eksponensial** — tangguh terhadap kesalahan jaringan sementara
- **Caching bawaan** — cache dalam memori dengan TTL yang dapat dikonfigurasi untuk menghindari kueri berlebihan
- **Backend cache kustom** — pasang Redis, memcached, atau backend apa pun melalui antarmuka `Cache`
- **Pemeriksaan kesehatan server** — pantau status online/offline dan latensi server DNS
- **Pemulihan panic** — goroutine dilindungi dari panic dengan pemulihan otomatis dan error yang diketik
- **Opsi fungsional** — pola konfigurasi Go yang bersih dan idiomatis
- **Sadar konteks** — dukungan penuh untuk timeout dan pembatalan melalui `context.Context`
- **Validasi domain** — normalisasi dan validasi nama domain otomatis
- **Error yang diketik** — error sentinel untuk pencocokan `errors.Is` (`ErrNoDNSServers`, `ErrAllDNSFailed`, `ErrInvalidDomain`, `ErrDNSTimeout`, `ErrInternalPanic`)

## Instalasi

```bash
go get github.com/H0llyW00dzZ/nawala-checker
```

Membutuhkan **Go 1.25.6** atau lebih baru.

## Mulai Cepat

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/H0llyW00dzZ/nawala-checker/src/nawala"
)

func main() {
    c := nawala.New()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    results, err := c.Check(ctx, "google.com", "reddit.com", "github.com")
    if err != nil {
        log.Fatal(err)
    }

    for _, r := range results {
        status := "tidak diblokir"
        if r.Blocked {
            status = "DIBLOKIR"
        }
        if r.Error != nil {
            status = fmt.Sprintf("error: %v", r.Error)
        }
        fmt.Printf("%-20s %s (server: %s)\n", r.Domain, status, r.Server)
    }
}
```

## Konfigurasi

Gunakan opsi fungsional untuk menyesuaikan pemeriksa:

```go
c := nawala.New(
    // Tingkatkan timeout untuk jaringan lambat.
    nawala.WithTimeout(15 * time.Second),

    // Izinkan lebih banyak percobaan ulang (3 percobaan ulang = 4 upaya total).
    nawala.WithMaxRetries(3),

    // Cache hasil selama 10 menit.
    nawala.WithCacheTTL(10 * time.Minute),

    // Ganti semua server DNS dengan server kustom.
    nawala.WithServers([]nawala.DNSServer{
        {
            Address:   "8.8.8.8",
            Keyword:   "blocked",
            QueryType: "A",
        },
        {
            Address:   "8.8.4.4",
            Keyword:   "blocked",
            QueryType: "A",
        },
    }),

    // Batasi pemeriksaan serentak hingga 50 goroutine.
    nawala.WithConcurrency(50),

    // Gunakan klien DNS kustom (misalnya, untuk TCP atau DNS-over-TLS).
    nawala.WithDNSClient(&dns.Client{
        Timeout: 10 * time.Second,
        Net:     "tcp-tls",
    }),

    // Atur ukuran EDNS0 kustom (default adalah 1232 untuk mencegah fragmentasi).
    nawala.WithEDNS0Size(4096),
)
```

### Pilihan Tersedia

| Opsi | Default | Deskripsi |
|---|---|---|
| `WithTimeout(d)` | `5s` | Timeout untuk setiap kueri DNS |
| `WithMaxRetries(n)` | `2` | Maksimum upaya percobaan balik per kueri (total = n+1) |
| `WithCacheTTL(d)` | `5m` | TTL untuk cache dalam memori bawaan |
| `WithCache(c)` | dalam-memori | Implementasi `Cache` kustom (pass `nil` untuk menonaktifkan) |
| `WithConcurrency(n)` | `100` | Maksimum pemeriksaan DNS serentak (ukuran semaphore) |
| `WithEDNS0Size(n)` | `1232` | Ukuran buffer UDP EDNS0 (mencegah fragmentasi) |
| `WithDNSClient(c)` | klien UDP | `*dns.Client` kustom untuk TCP, TLS, atau dialer kustom |
| `WithServer(s)` | — | Tambahkan atau ganti satu server DNS |
| `WithServers(s)` | Default Nawala | Ganti semua server DNS |

## API

### Metode Inti

```go
// Periksa beberapa domain secara serentak.
results, err := c.Check(ctx, "example.com", "another.com")

// Periksa satu domain.
result, err := c.CheckOne(ctx, "example.com")

// Periksa kesehatan dan latensi server DNS.
statuses, err := c.DNSStatus(ctx)

// Bersihkan cache hasil.
c.FlushCache()

// Dapatkan server yang dikonfigurasi.
servers := c.Servers()
```

### Validasi

```go
// Validasi nama domain sebelum memeriksa.
ok := nawala.IsValidDomain("example.com") // true
ok  = nawala.IsValidDomain("invalid")     // false (satu label, tidak ada TLD)
```

### Tipe

```go
// Hasil pemeriksaan satu domain.
type Result struct {
    Domain  string  // Domain yang diperiksa
    Blocked bool    // Apakah domain diblokir
    Server  string  // IP server DNS yang digunakan untuk pemeriksaan
    Error   error   // Non-nil jika pemeriksaan gagal
}

// Status kesehatan server DNS.
type ServerStatus struct {
    Server    string  // Alamat IP server DNS
    Online    bool    // Apakah server merespons
    LatencyMs int64   // Waktu pulang pergi dalam milidetik
    Error     error   // Non-nil jika pemeriksaan kesehatan gagal
}

// Konfigurasi server DNS.
type DNSServer struct {
    Address   string  // Alamat IP server DNS
    Keyword   string  // Kata kunci pemblokiran untuk dicari dalam respons
    QueryType string  // Tipe record DNS: "A", "AAAA", "CNAME", "TXT", dll.
}
```

### Error

```go
var (
    ErrNoDNSServers  // Tidak ada server DNS yang dikonfigurasi
    ErrAllDNSFailed  // Semua server DNS gagal merespons
    ErrInvalidDomain // Nama domain gagal validasi
    ErrDNSTimeout    // Kueri DNS melebihi timeout yang dikonfigurasi
    ErrInternalPanic // Panic internal dipulihkan selama eksekusi
)
```

### Cache Kustom

Implementasikan antarmuka `Cache` untuk menggunakan backend cache kustom:

```go
type Cache interface {
    Get(key string) (Result, bool)
    Set(key string, val Result)
    Flush()
}
```

## Contoh

Contoh yang dapat dijalankan tersedia di direktori [`examples/`](examples/):

| Contoh | Deskripsi |
|---|---|
| [`basic`](examples/basic) | Periksa beberapa domain dengan konfigurasi default |
| [`custom`](examples/custom) | Konfigurasi lanjutan dengan server kustom, timeout, percobaan ulang, dan caching |
| [`status`](examples/status) | Pantau kesehatan dan latensi server DNS |

Jalankan contoh:

```bash
go run ./examples/basic
```

## Server DNS Default

Pemeriksa datang dengan pra-konfigurasi server DNS Nawala yang dikenal:

| Server | Kata Kunci | Tipe Kueri |
|---|---|---|
| `180.131.144.144` | `internetpositif` | `A` |
| `180.131.145.145` | `internetpositif` | `A` |

Nawala memblokir domain dengan mengembalikan pengalihan CNAME ke halaman blokir yang dikenal (`internetpositif.id` atau `internetsehatku.com`). Komdigi memblokir domain dengan mengembalikan record A dengan EDE 15 (Blocked) yang berisi `trustpositif.komdigi.go.id`. Kata kunci dicocokkan dengan seluruh string record DNS untuk deteksi luas.

## Bagaimana Pemblokiran Bekerja

Filter DNS ISP Indonesia menggunakan dua mekanisme pemblokiran yang berbeda:

### Nawala — Pengalihan CNAME

Nawala memotong kueri DNS untuk domain yang diblokir dan mengembalikan **pengalihan CNAME** ke halaman pendaratan alih-alih alamat IP asli:

```
;; ANSWER SECTION:
blocked.example.    3600    IN    CNAME    internetpositif.id.
```

Pemeriksa mendeteksi ini dengan memindai semua bagian record DNS (Answer, Authority, Additional) untuk kata kunci `internetpositif` dalam representasi string record apa pun.

### Komdigi — EDE 15 (Blocked)

Komdigi menggunakan mekanisme **Extended DNS Errors** yang lebih baru ([RFC 8914](https://datatracker.ietf.org/doc/rfc8914/)). Respons mengembalikan record A yang menunjuk ke IP halaman blokir, bersama dengan kode opsi EDE 15 (Blocked) di bagian pseudo OPT:

```
;; OPT PSEUDOSECTION:
; EDE: 15 (Blocked): (source=block-list-zone;
;   blockListUrl=https://trustpositif.komdigi.go.id/assets/db/domains_isp;
;   domain=reddit.com)

;; ANSWER SECTION:
reddit.com.    30    IN    A    103.155.26.29
```

Pemeriksa mendeteksi ini dengan memindai bagian Extra (yang berisi record OPT) untuk kata kunci `trustpositif` atau `komdigi`. Untuk menggunakan deteksi ini, konfigurasikan server dengan kata kunci yang sesuai:

```go
nawala.WithServers([]nawala.DNSServer{
    {
        Address:   "103.155.26.28",
        Keyword:   "trustpositif",
        QueryType: "A",
    },
    {
        Address:   "103.155.26.29",
        Keyword:   "komdigi",
        QueryType: "A",
    },
})
```

## Struktur Proyek

```
nawala-checker/
├── .github/            # Alur kerja CI dan konfigurasi Dependabot
├── examples/           # Contoh penggunaan yang dapat dijalankan (basic, custom, status)
├── Makefile            # Pintasan build dan test
└── src/
    └── nawala/          # Paket SDK inti (checker, cache, DNS, options, types)
```

## Pengujian

```bash
# Jalankan tes dengan detektor race.
make test

# Jalankan tes dengan output verbose.
make test-verbose

# Jalankan tes dengan laporan cakupan.
make test-cover

# Lewati tes DNS langsung.
make test-short
```

## Peta Jalan

- [ ] Tingkatkan `github.com/miekg/dns` ke v2 atau gunakan alternatif modern untuk meningkatkan kinerja dan fitur jaringan, karena implementasinya di Go dan efektivitasnya yang tinggi untuk jaringan.

## Lisensi

[BSD 3-Clause License](LICENSE) — Hak Cipta (c) 2026, H0llyW00dzZ
