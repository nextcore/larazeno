# Panduan Penggunaan LaraZeno-Go di Windows

Panduan ini menjelaskan cara mengompilasi, melakukan migrasi database, dan menjalankan aplikasi Laravel di Windows menggunakan **LaraZeno-Go** (Zero Composer, Zero Vendor, Pure Go).

---

## 1. Prasyarat

- **Go 1.22+** terinstall di sistem Windows.
  ```powershell
  go version
  ```
- *Catatan*: **Tidak membutuhkan** PHP CLI, Composer, maupun compiler C/C++ (GCC/MinGW) karena driver SQLite sudah menggunakan pure Go (`modernc.org/sqlite`).

---

## 2. Compile Binary `larazeno.exe`

Buka PowerShell di folder `larazeno-go`:

```powershell
cd d:\Z\GH\larazeno\larazeno-go
go build -o larazeno.exe .
```

Binary `larazeno.exe` akan terbuat di folder `larazeno-go`.

---

## 3. Persiapan Target Proyek Laravel

Pastikan proyek Laravel target (misal `examples\laravel-official`) sudah memiliki file `.env` dan `database.sqlite`:

```powershell
cd d:\Z\GH\larazeno\examples\laravel-official

# 1. Salin template .env jika belum ada
if (!(Test-Path .env)) { Copy-Item .env.example .env }

# 2. Buat file SQLite kosong jika belum ada
if (!(Test-Path database\database.sqlite)) { New-Item -ItemType File database\database.sqlite }
```

---

## 4. Cara Menjalankan (Pilih Salah Satu)

LaraZeno mendukung **path relatif**, **path absolut**, maupun eksekusi langsung dari dalam folder proyek target.

### Opsi A: Dari Folder `larazeno-go` (Menggunakan Path Relatif)

```powershell
cd d:\Z\GH\larazeno\larazeno-go

# 1. Cek status migrasi
.\larazeno.exe -path "..\examples\laravel-official" migrate:status

# 2. Jalankan migrasi database
.\larazeno.exe -path "..\examples\laravel-official" migrate

# 3. Jalankan HTTP server (default port 9001)
.\larazeno.exe -path "..\examples\laravel-official" -port 9001
```

---

### Opsi B: Langsung dari Dalam Folder Proyek Laravel

Jika terminal Anda berada di dalam folder proyek Laravel, Anda **tidak perlu menyertakan flag `-path`**:

```powershell
cd d:\Z\GH\larazeno\examples\laravel-official

# 1. Cek status migrasi
..\..\larazeno-go\larazeno.exe migrate:status

# 2. Jalankan migrasi database
..\..\larazeno-go\larazeno.exe migrate

# 3. Jalankan HTTP server
..\..\larazeno-go\larazeno.exe -port 9001
```

---

## 5. Menguji Aplikasi

Setelah server berjalan (`LaraZeno Server listening on http://127.0.0.1:9001`), buka browser atau terminal baru:

| Kebutuhan | URL / Perintah |
| :--- | :--- |
| **Halaman Blade Laravel** | Buka browser ke `http://127.0.0.1:9001/` atau `curl.exe -i http://127.0.0.1:9001/` |
| **REST API Produk (JSON)** | `curl.exe -i http://127.0.0.1:9001/api/products` |
| **REST API Orders (JSON)** | `curl.exe -i http://127.0.0.1:9001/api/pos/orders` |
| **Halaman POS (Web View)** | Buka browser ke `http://127.0.0.1:9001/pos` |

---

## 6. Tips Tambahan

1. **Mengganti Port**:
   Jika port `9001` bentrok, tentukan port lain via `-port`, contoh:
   ```powershell
   .\larazeno.exe -path "..\examples\laravel-official" -port 8080
   ```
2. **Koneksi MySQL/MariaDB**:
   Ubah pengaturan `DB_CONNECTION=mysql` dan sesuaikan kredensial di file `.env` proyek Laravel Anda. LaraZeno akan otomatis mendeteksi konfigurasi tersebut.
