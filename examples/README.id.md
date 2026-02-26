# Contoh

[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/nawala-checker.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/nawala-checker)
[![Read in English](https://img.shields.io/badge/🇬🇧-Read%20in%20English-blue)](README.md)

Direktori ini berisi contoh yang dapat dijalankan secara langsung untuk SDK
[nawala-checker](https://github.com/H0llyW00dzZ/nawala-checker) — sebuah
pemeriksa pemblokiran domain berbasis DNS untuk filter DNS ISP Indonesia
(Nawala/Kominfo, sekarang Komdigi).

> [!IMPORTANT]
> Contoh-contoh ini memerlukan **jaringan Indonesia** agar menghasilkan hasil
> pemblokiran yang bermakna. Server DNS Nawala dan Komdigi hanya mengembalikan
> indikator blokir saat dikueri dari alamat IP Indonesia. Jika Anda menjalankan
> dari luar Indonesia, konfigurasikan server DNS kustom yang dihosting di
> jaringan Indonesia dan arahkan checker ke sana melalui `WithServers`.

| Contoh | Deskripsi |
|---|---|
| [`basic/`](basic/main.go) | Periksa beberapa domain dengan konfigurasi default |
| [`custom/`](custom/main.go) | Konfigurasi lanjutan: server kustom, timeout, percobaan ulang, caching |
| [`status/`](status/main.go) | Pantau kesehatan dan latensi server DNS |
| [`hotreload/`](hotreload/main.go) | Perbarui server DNS dengan aman secara konkurensi saat pemeriksaan berjalan |

## Prasyarat

- **Go 1.25.6** atau lebih baru
- **Repositori yang telah dikloning** — contoh tidak didistribusikan melalui `go get`:
  ```bash
  git clone https://github.com/H0llyW00dzZ/nawala-checker.git
  cd nawala-checker
  ```
- Koneksi **jaringan Indonesia** (atau relay DNS kustom di jaringan Indonesia —
  lihat tips di bawah)

> [!TIP]
> Saat berjalan pada infrastruktur cloud (misalnya, VPS, microservice, [k8s](https://kubernetes.io)) yang
> tidak berada pada jaringan Indonesia (misalnya, server Singapura atau AS),
> implementasikan server DNS Anda sendiri di jaringan Indonesia, kemudian arahkan
> SDK ini ke sana menggunakan `WithServers`. Perilaku indikator pemblokiran bergantung
> pada server DNS yang digunakan; server Nawala/Komdigi default hanya akan
> mengembalikan indikator pemblokiran saat diquery dari IP sumber Indonesia.

## Menjalankan Contoh

```bash
go run ./examples/basic
go run ./examples/custom
go run ./examples/status
go run ./examples/hotreload
```

---

## `basic` — Konfigurasi Default

[`basic/main.go`](basic/main.go) memeriksa sejumlah domain terhadap server
DNS Nawala yang sudah dikonfigurasi sebelumnya tanpa pengaturan manual apa pun.

```go
c := nawala.New()

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := c.Check(ctx, "google.com", "reddit.com", "github.com", "exam_ple.com")
if err != nil {
    log.Fatalf("check failed: %v", err)
}

for _, r := range results {
    status := "tidak diblokir"
    if r.Blocked {
        status = "DIBLOKIR"
    }
    if r.Error != nil {
        status = fmt.Sprintf("error: %v", r.Error)
    }
    fmt.Printf("  %-20s %s (server: %s)\n", r.Domain, status, r.Server)
}
```

**Keluaran yang diharapkan** (dari jaringan Indonesia):

```
=== Nawala DNS Blocker Check ===

  exam_ple.com         error: nawala: nxdomain: domain does not exist (NXDOMAIN) (server: 180.131.144.144)
  google.com           tidak diblokir (server: 180.131.144.144)
  reddit.com           DIBLOKIR (server: 180.131.144.144)
  github.com           tidak diblokir (server: 180.131.144.144)
```

**Yang ditunjukkan contoh ini:**

- `nawala.New()` tanpa opsi menggunakan server DNS Nawala bawaan
  (`180.131.144.144`, `180.131.145.145`)
- `Check` memeriksa semua domain secara **bersamaan** dan mengembalikan satu
  `Result` per domain
- `Result.Blocked` bernilai `true` saat respons DNS mengandung kata kunci
  pemblokiran (misalnya pengalihan CNAME ke `internetpositif.id`)
- `Result.Error` bernilai non-nil saat pemeriksaan itu sendiri gagal (kesalahan
  jaringan, timeout, dll.) — terpisah dari status pemblokiran

---

## `custom` — Konfigurasi Lanjutan

[`custom/main.go`](custom/main.go) menunjukkan cara menggunakan opsi fungsional
untuk menyetel checker, menambahkan server DNS tambahan, dan mendemonstrasikan
cache bawaan.

```go
c := nawala.New(
    // Tambahkan server DNS kustom bersama server default.
    nawala.WithServer(nawala.DNSServer{
        Address:   "8.8.8.8",
        Keyword:   "blocked",
        QueryType: "A",
    }),

    // Tingkatkan timeout untuk jaringan lambat.
    nawala.WithTimeout(15 * time.Second),

    // Izinkan lebih banyak percobaan ulang (3 percobaan = 4 upaya total).
    nawala.WithMaxRetries(3),

    // Cache hasil selama 10 menit.
    nawala.WithCacheTTL(10 * time.Minute),
)
```

**Keluaran yang diharapkan** (dari jaringan Indonesia):

```
=== Custom Configuration Check ===

Configured DNS servers:
  180.131.144.144 (keyword="internetpositif", type=A)
  180.131.145.145 (keyword="internetpositif", type=A)
  8.8.8.8 (keyword="blocked", type=A)

  google.com: tidak diblokir (server: 180.131.144.144)

=== Runtime Reconfiguration ===

Adding new server 203.0.113.1 at runtime...
Successfully verified 203.0.113.1 is active!
Removing server 203.0.113.1...
Successfully verified 203.0.113.1 was removed!

Second check (cached):
  google.com: blocked=false (took 2.6µs)
```

**Yang ditunjukkan contoh ini:**

- `WithServer` **menambahkan** satu server ke daftar yang ada (Usang: gunakan
  `c.SetServers()` untuk **mengganti** atau menambah server saat runtime
  dengan aman secara konkurensi)
- `c.SetServers()` dan `c.DeleteServers()` memungkinkan *hot-reload* dinamis dari server DNS.
- `c.HasServer()` memverifikasi apakah IP server DNS tertentu saat ini dikonfigurasi.
- `WithTimeout` dan `WithMaxRetries` mengontrol ketahanan per kueri
- `WithCacheTTL` mengaktifkan cache TTL dalam memori — panggilan `CheckOne`
  kedua selesai dalam milidetik karena hasilnya disajikan dari cache
- `c.Servers()` mengembalikan daftar lengkap server DNS yang dikonfigurasi
  pada saat runtime

---

## `status` — Pemeriksaan Kesehatan Server DNS

[`status/main.go`](status/main.go) mengkueri semua server DNS yang dikonfigurasi
dan melaporkan status online/offline serta latensi pulang-pergi masing-masing.

```go
c := nawala.New()

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

statuses, err := c.DNSStatus(ctx)
if err != nil {
    log.Fatalf("status check failed: %v", err)
}

for _, s := range statuses {
    status := "OFFLINE"
    if s.Online {
        status = fmt.Sprintf("ONLINE (%dms)", s.LatencyMs)
    }
    fmt.Printf("  %-18s %s\n", s.Server, status)
    if s.Error != nil {
        fmt.Printf("    error: %v\n", s.Error)
    }
}
```

**Keluaran yang diharapkan** (dari jaringan Indonesia):

```
=== Nawala DNS Server Status ===

  180.131.144.144    ONLINE (12ms)
  180.131.145.145    ONLINE (14ms)
```

**Yang ditunjukkan contoh ini:**

- `DNSStatus` memeriksa semua server yang dikonfigurasi dan mengembalikan satu
  `ServerStatus` per server
- `ServerStatus.Online` bernilai `true` saat server merespons probe kesehatan
  dalam timeout yang dikonfigurasi
- `ServerStatus.LatencyMs` adalah `int64` dari **milidetik bulat** (misalnya `12`
  berarti 12 ms); hanya diisi saat `Online` bernilai `true` — server yang
  offline meninggalkannya sebagai `0`
- `ServerStatus.Error` bernilai non-nil saat probe kesehatan itu sendiri gagal
- Berguna untuk pemantauan atau pemeriksaan awal sebelum menjalankan
  pemeriksaan domain secara massal

---

## `hotreload` — Pembaruan Konfigurasi Aman Konkurensi

[`hotreload/main.go`](hotreload/main.go) menjalankan pemeriksaan DNS terus-menerus
dalam loop sambil secara bersamaan menggunakan `SetServers` untuk mengubah 
server DNS dan kata kuncinya di latar belakang.

**Keluaran yang diharapkan** (waktu bervariasi):

```
=== Nawala DNS Hot-Reload Example ===
[15:04:05.105] reddit.com      -> tidak diblokir (Server: 8.8.8.8, Keyword: this-will-never-match)
[15:04:05.619] reddit.com      -> tidak diblokir (Server: 8.8.8.8, Keyword: this-will-never-match)

>>> TRIGGERING HOT-RELOAD: Adding Nawala Block Server...
[15:04:07.135] reddit.com      -> DIBLOKIR     (Server: 180.131.144.144, Keyword: internetpositif)
[15:04:07.651] reddit.com      -> DIBLOKIR     (Server: 180.131.144.144, Keyword: internetpositif)

>>> TRIGGERING HOT-RELOAD: Changing Keyword...
[15:04:10.165] reddit.com      -> tidak diblokir (Server: 180.131.144.144, Keyword: changed-keyword)

>>> TRIGGERING HOT-RELOAD: Deleting Server...
[15:04:12.670] reddit.com      -> Error: nawala: no DNS servers configured
```

**Yang ditunjukkan contoh ini:**

- `c.SetServers(...)` memperoleh kunci eksklusif secara internal untuk mengganti
  semen server.
- `c.DeleteServers(...)` memperoleh kunci eksklusif secara internal untuk menghapus 
  server melalui alamat IP mereka. Jika semua server dihapus, pemeriksaan bersamaan
  dengan aman melakukan short-circuit dan mengembalikan `ErrNoDNSServers`.
- `c.CheckOne()` memperoleh kunci baca cepat untuk menyalin konfigurasi saat ini,
  memastikan kode tidak pernah panic pada *race condition*.
- Anda dapat sepenuhnya menimpa properti server yang ada (seperti `Keyword`
  atau `QueryType`) jika meneruskan konfigurasi server dengan `Address` yang identik.
