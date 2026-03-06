# 🌐 Nawala Checker

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.25.6-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Go Report Card](https://goreportcard.com/badge/github.com/H0llyW00dzZ/nawala-checker)](https://goreportcard.com/report/github.com/H0llyW00dzZ/nawala-checker)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/H0llyW00dzZ/nawala-checker/graph/badge.svg?token=3GU9QRYAUX)](https://codecov.io/gh/H0llyW00dzZ/nawala-checker)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/H0llyW00dzZ/nawala-checker)
[![Read in English](https://img.shields.io/badge/🇬🇧-Read%20in%20English-blue)](README.md)

SDK Go untuk memeriksa apakah domain diblokir oleh filter DNS ISP Indonesia (Nawala/Kominfo (sekarang Komdigi)). SDK ini bekerja dengan menanyakan server DNS yang dapat dikonfigurasi dan memindai respons untuk kata kunci pemblokiran seperti pengalihan `internetpositif.id` atau indikator EDE `trustpositif.komdigi.go.id`.

> [!IMPORTANT]
> SDK ini membutuhkan **jaringan Indonesia** agar berfungsi dengan benar. Server DNS Nawala hanya mengembalikan respons pemblokiran saat ditanyai dari dalam Indonesia. Pastikan koneksi Anda menggunakan IP Indonesia murni tanpa perutean melalui jaringan di luar Indonesia.

> [!NOTE]
> **SDK ini tidak usang atau ketinggalan zaman.** Meskipun ada rumor bahwa proyek Nawala yang asli mungkin berhenti beroperasi, modul ini tetap menjadi **perangkat pemeriksaan DNS tujuan umum** yang dibangun dari dasar dengan konfigurasi server dan klien DNS yang dapat disesuaikan. Anda dapat mengarahkannya ke server DNS mana pun, menentukan kata kunci pemblokiran Anda sendiri, dan memasang instance `*dns.Client` kustom (TCP, DNS-over-TLS, dialer kustom, dll.). Server Nawala default hanyalah default yang sudah dikonfigurasi sebelumnya; SDK itu sendiri sepenuhnya independen dan dipelihara secara aktif.

> [!TIP]
> Saat berjalan pada infrastruktur cloud (misalnya, VPS, microservice, [k8s](https://kubernetes.io)) yang tidak berada pada jaringan Indonesia (misalnya, server Singapura atau AS), implementasikan server DNS Anda sendiri di jaringan Indonesia, kemudian arahkan SDK ini ke sana menggunakan `WithServers`. Perilaku indikator pemblokiran bergantung pada server DNS yang digunakan; server Nawala/Komdigi default hanya mengembalikan indikator pemblokiran saat diquery dari IP sumber Indonesia.

## Fitur

- **Pemeriksaan domain serentak** — periksa beberapa domain secara paralel dengan satu panggilan
- **Failover server DNS** — fallback otomatis ke server sekunder ketika server utama gagal
- **Coba lagi dengan backoff eksponensial** — tangguh terhadap kesalahan jaringan sementara
- **Caching bawaan** — cache dalam memori dengan TTL yang dapat dikonfigurasi untuk menghindari kueri berlebihan
- **Backend cache kustom** — pasang Redis, memcached, atau backend apa pun melalui antarmuka `Cache`
- **Pemeriksaan kesehatan server** — pantau status online/offline dan latensi server DNS
- **Pemulihan panic** — goroutine dilindungi dari panic dengan pemulihan otomatis dan error yang diketik
- **Opsi fungsional** — pola konfigurasi Go yang bersih dan [idiomatis](https://go.dev/doc/effective_go)
- **Sadar konteks** — dukungan penuh untuk timeout dan pembatalan melalui `context.Context`
- **Validasi domain** — normalisasi dan validasi nama domain otomatis
- **Error yang diketik** — error sentinel untuk pencocokan `errors.Is` (lihat [Error](#error))

## 🚀 Performa

Karena SDK ini ditulis dalam Go dan beroperasi di **level protokol DNS** (bukan melalui REST API), SDK ini menghindari overhead HTTP seperti serialisasi JSON, TLS handshake per-request, dan batas multipleksing HTTP/2 yang memengaruhi pendekatan berbasis API. Setiap kueri DNS adalah paket UDP atau TCP yang ringkas — biasanya di bawah 512 byte — sehingga biaya per-domain sangat rendah.

Konkurensi dikelola oleh **semaphore channel buffer** (`WithConcurrency`, default 100). Setiap domain dikirim ke goroutine-nya sendiri, dan semaphore memastikan sistem tidak pernah membuat goroutine tak terbatas. Model ini berskala secara linear: naikkan batasnya dan SDK akan secara otomatis menggunakan lebih banyak core.

| Pendekatan | Overhead protokol | Serialisasi | Model konkurensi |
|---|---|---|---|
| **SDK ini** | DNS over UDP/TCP (~64 B per kueri) | Tidak ada — format wire DNS biner | Goroutine per domain, dibatasi semaphore |
| Checker REST API | HTTP/HTTPS + JSON | Encode/decode JSON per request | Umumnya dibatasi request-pool |

Dalam praktiknya, SDK ini mampu memeriksa **jutaan — bahkan miliaran — domain** dalam satu kali eksekusi. Batas atas ditentukan oleh kecepatan respons server DNS, bukan SDK itu sendiri:

- **Memori** — setiap goroutine membawa stack kecil (~2–8 KB dasar); 100 goroutine concurrent ≈ < 1 MB overhead
- **`WithConcurrency(n)`** — naikkan `n` untuk menyesuaikan kapasitas server; turunkan untuk tidak membebani resolver bersama
- **Tidak ada batas jumlah domain** — SDK mengirim dan memproses domain sesuai permintaan

> [!TIP]
> Untuk daftar domain yang sangat besar (jutaan hingga miliaran), kombinasikan nilai `WithConcurrency` yang tinggi dengan `WithCache` dinonaktifkan (atau cache berbasis Redis) dan streaming domain dari file menggunakan `--file` di CLI.

## Instalasi

```bash
go get github.com/H0llyW00dzZ/nawala-checker
```

Membutuhkan **Go 1.25.6** atau lebih baru.

### CLI

Instal alat baris perintah `nawala`:

```bash
go install github.com/H0llyW00dzZ/nawala-checker/cmd/nawala@latest
```

Penggunaan:

```bash
# Periksa domain (singkatan — mendelegasikan ke "check")
nawala google.com reddit.com

# Periksa domain dari file
nawala check --file domains.txt

# Output JSON (NDJSON — satu objek per baris)
nawala check google.com --format json

# Tulis hasil ke file (teks dipisahkan tab)
nawala check --file domains.txt -o results.txt

# Buat laporan HTML
nawala check google.com reddit.com --format html -o report.html

# Buat spreadsheet Excel (XLSX)
nawala check --file domains.txt --format xlsx -o results.xlsx

# Gunakan konfigurasi kustom (JSON atau YAML)
nawala check --config config.json --file domains.txt

# Periksa konfigurasi efektif (tampilkan semua default)
nawala config
nawala config --json

# Buat file konfigurasi
nawala config -o myconfig.json --json

# Tampilkan kesehatan dan latensi server DNS
nawala status

# Tampilkan versi
nawala --version
```

> [!NOTE]
> Input domain bersifat **case-insensitive** — `Google.com` dan `google.com` dianggap sebagai
> domain yang sama dan akan dideduplikasi sebelum diperiksa. Domain Unicode (IDN) secara
> **otomatis dikonversi** ke Punycode (ACE) menggunakan profil IDNA Lookup (UTS\#46), sehingga
> Anda dapat memasukkan `例え.jp` langsung dan akan diperiksa sebagai `xn--r8jz45g.jp`.

Contoh file konfigurasi (`config.json`) — format nawala envelope:

```json
{
  "nawala": {
    "version": "0.6.5",
    "configuration": {
      "timeout": "5s",
      "command_timeout": "30s",
      "max_retries": 2,
      "cache_ttl": "5m",
      "disable_cache": false,
      "concurrency": 100,
      "edns0_size": 1232,
      "protocol": "udp",
      "tls_server_name": "",
      "tls_skip_verify": false,
      "servers": [
        {"address": "180.131.144.144", "keyword": "internetpositif", "query_type": "A"},
        {"address": "103.155.26.28",  "keyword": "trustpositif",    "query_type": "A"}
      ]
    }
  }
}
```

> [!NOTE]
> Field `version` mencatat versi CLI yang menghasilkan konfigurasi. Jika tidak cocok dengan
> CLI yang berjalan, peringatan akan dicetak ke stderr dan konfigurasi tetap diterapkan.
> Buat ulang dengan `nawala config --json -o config.json` untuk memperbaruinya.
>
> Atur `disable_cache: true` untuk menonaktifkan cache dalam memori bawaan sepenuhnya.
> Jika diaktifkan, nilai `cache_ttl` tidak berpengaruh.
>
> Atur `edns0_size` untuk mengontrol ukuran buffer UDP EDNS0 (default `1232`, atur ke `4096`
> untuk resolver yang mendukung payload yang lebih besar).
>
> Atur `protocol` ke `"udp"` (default), `"tcp"`, atau `"tcp-tls"` (DNS over TLS / DoT)
> untuk memilih transport DNS. Timeout dan ukuran EDNS0 tetap berlaku di semua protokol.
>
> Untuk `tcp-tls`, dua field TLS opsional tersedia:
> - `tls_server_name` — mengganti nama SNI TLS (berguna saat alamat server berupa IP)
> - `tls_skip_verify` — menonaktifkan verifikasi sertifikat; hanya untuk sertifikat self-signed, jangan digunakan di production


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

    results, err := c.Check(ctx, "google.com", "reddit.com", "github.com", "exam_ple.com")
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

    // Gunakan transport DNS-over-TLS (DoT).
    nawala.WithProtocol("tcp-tls"),
    nawala.WithTLSServerName("dns.example.com"),

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
| `WithProtocol(s)` | `"udp"` | Transport DNS: `"udp"`, `"tcp"`, atau `"tcp-tls"` (DoT) |
| `WithTLSServerName(s)` | `""` | Override nama server TLS SNI (hanya tcp-tls) |
| `WithTLSSkipVerify()` | `false` | Lewati verifikasi sertifikat TLS (hanya tcp-tls) |
| `WithDNSClient(c)` | klien UDP | `*dns.Client` kustom untuk TCP, TLS, atau dialer kustom |
| `WithServer(s)` | — | **Usang (Deprecated):** gunakan `Checker.SetServers`. Tambahkan atau ganti server tunggal |
| `WithServers(s)` | Default Nawala | Ganti semua server DNS |
| `Checker.SetServers(s)` | — | Hot-reload: Tambahkan atau ganti server saat runtime (aman untuk konkurensi) |
| `Checker.HasServer(s)` | — | Hot-reload: Periksa apakah server dikonfigurasi saat runtime (aman untuk konkurensi) |
| `Checker.DeleteServers(s)` | — | Hot-reload: Hapus server saat runtime (aman untuk konkurensi) |

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

// Hot-reload: Tambahkan atau ganti server saat runtime (aman untuk konkurensi).
c.SetServers(nawala.DNSServer{
    Address:   "203.0.113.1",
    Keyword:   "blocked",
    QueryType: "A",
})

// Hot-reload: Periksa apakah server saat ini dikonfigurasi.
if c.HasServer("203.0.113.1") {
    fmt.Println("Server is active")
}

// Hot-reload: Hapus server melalui alamat IP saat runtime (aman untuk konkurensi).
c.DeleteServers("203.0.113.1")
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
    ErrNXDOMAIN      // Domain tidak ada (NXDOMAIN)
    ErrQueryRejected // Kueri secara eksplisit ditolak oleh server (Format Error, Refused, Not Implemented)
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
| [`hotreload`](examples/hotreload) | Perbarui server DNS dengan aman secara konkurensi saat pemeriksaan berjalan |

Jalankan contoh (membutuhkan kloning repositori):

```bash
git clone https://github.com/H0llyW00dzZ/nawala-checker.git
cd nawala-checker
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

## 📜 Legenda Nawala

Bagi banyak pengguna internet jadul (generasi anak warnet), **DNS Nawala** adalah nama yang sangat legendaris. Mengambil nama dari bahasa Jawa Kuno yang berarti "surat" atau "pesan", Proyek Nawala bermula sebagai inisiatif para aktivis internet sekitar tahun 2007-2009. Layanan DNS gratis ini dirancang untuk menapis konten negatif (pornografi, perjudian, dan *malware*) demi menciptakan internet yang sehat dan aman di Indonesia. Jauh sebelum istilah *Internet Positif* menjadi populer, jika Anda tidak dapat mengakses sebuah situs, kemungkinan besar Anda sedang diblokir oleh Nawala.

Sistem ini dulunya sangat umum ditemukan di warung internet (*warnet*) dan ISP awal, sehingga mencari cara untuk menembus Nawala melalui server DNS kustom (seperti `8.8.8.8` milik Google) terasa seperti sebuah ritual inisiasi bagi netizen Indonesia. Seiring berjalannya waktu, Nawala telah bertransformasi dari berfokus pada keamanan internet, ke pengembangan aplikasi, hingga kini berfokus pada pelatihan dan pendidikan teknologi untuk memberikan kontribusi sosial (dikenal sebagai Nawala Education).

Hari ini, meskipun layanan DNS filtering Nawala yang asli mungkin sudah menjadi sejarah, warisannya tetap hidup. Pemerintah Indonesia (Kominfo, sekarang Komdigi) mengadopsi dan memperluas konsep-konsep tersebut, berkembang dari pengalihan CNAME awal (`internetpositif.id`) ke model *Extended DNS Errors* modern yang sesuai standar (`trustpositif.komdigi.go.id`). SDK ini menghormati sejarah tersebut sekaligus menyediakan alat yang kuat untuk menavigasi lanskap penyaringan internet Indonesia era modern.

## Struktur Proyek

```
nawala-checker/
├── .github/            # Alur kerja CI dan konfigurasi Dependabot
├── cmd/
│   └── nawala/         # Titik masuk CLI
├── examples/           # Contoh penggunaan yang dapat dijalankan (basic, custom, status)
├── internal/
│   └── cli/            # Paket CLI (perintah, konfigurasi, output)
├── Makefile            # Pintasan build dan test
└── src/
    └── nawala/         # Paket SDK inti (checker, cache, DNS, options, types)
```

## Pengujian

Pengujian harus dijalankan dari repositori yang telah dikloning:

```bash
git clone https://github.com/H0llyW00dzZ/nawala-checker.git
cd nawala-checker
```

Kemudian jalankan target yang diinginkan:

```bash
# Jalankan tes dengan detektor race.
make test

# Jalankan tes dengan output verbose.
make test-verbose

# Jalankan tes dengan laporan cakupan.
make test-cover

# Lewati tes DNS langsung.
make test-short

# Bangun biner CLI.
make build
```

## Peta Jalan

- [ ] Tingkatkan `github.com/miekg/dns` ke v2 atau gunakan alternatif modern untuk meningkatkan kinerja dan fitur jaringan, karena implementasinya di Go dan efektivitasnya yang tinggi untuk jaringan.
- [x] Implementasikan versi CLI (dibundel dalam repositori ini) untuk memeriksa domain langsung dari terminal tanpa perlu menulis kode Go.
- [ ] Implementasikan versi server [MCP](https://modelcontextprotocol.io/docs/getting-started/intro) ([Model Context Protocol](https://modelcontextprotocol.io/docs/getting-started/intro)) (dibundel dalam repositori ini) untuk mengintegrasikan nawala-checker secara langsung dengan agen AI dan LLM.
- [ ] Implementasikan versi server [JSON-RPC 2.0](https://www.jsonrpc.org/specification) murni (dibundel dalam repositori ini) untuk integrasi lintas bahasa melalui stdio atau TCP, serupa cara kerja MCP namun menggunakan protokol kawat JSON-RPC standar.

## Lisensi

[BSD 3-Clause License](LICENSE) — Hak Cipta (c) 2026, H0llyW00dzZ
