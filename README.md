# LaraZeno

**LaraZeno** adalah *high-performance Laravel execution engine* yang ditulis menggunakan bahasa Go murni. LaraZeno mem-parse kode sumber Laravel secara langsung menjadi **Abstract Syntax Tree (AST)** dan mengeksekusinya tanpa memerlukan instalasi PHP, Composer, maupun folder fisik `vendor/` di lingkungan host (*Zero PHP Runtime, Zero Composer, Zero Vendor Folder*).

---

## ⚡ Fitur Utama

- **Zero Composer & Zero Vendor Folder**: Tidak perlu `composer install` atau memelihara folder `vendor` yang memakan ruang ratusan megabyte. LaraZeno memetakan dependensi via *Virtual Vendor Provider*.
- **Pure Go PHP Lexer & AST Parser**: Mengurai sintaks controller, routes, dan model Laravel langsung ke representasi AST di memori.
- **Universal REST API Engine**:
  - Auto-discovery route (`routes/api.php`, `routes/web.php`, dll).
  - Mendukung `Route::get`, `Route::post`, `Route::put`, `Route::delete`.
  - Mendukung `Route::apiResource('endpoint', Controller::class)`.
  - Penanganan parameter dinamis URL (e.g. `/{id}`).
- **Built-in Authentication (Laravel Sanctum)**:
  - Dukungan middleware `auth:sanctum`.
  - Validasi *Bearer Token* dan resolusi `$request->user()`.
- **Lightweight Eloquent ORM Emulation**:
  - Operasi dasar: `Model::all()`, `Model::find($id)`, `Model::create($data)`, `$model->delete()`.
  - Mendukung driver database **MySQL / MariaDB** dan **SQLite** sesuai konfigurasi `.env`.
- **Blade Template Engine (Go Native)**:
  - Parsing direktif `@if`, `@foreach`, `@include`, dan interpolasi variabel `{{ $var }}`.
- **Security Guard**:
  - Proteksi terintegrasi dari akses file `.env`, webshell, atau eksekusi skrip ilegal langsung.

---

## 📁 Struktur Direktori

```text
larazeno/
├── main.go                     # Entry point CLI & HTTP Dispatcher
├── go.mod / go.sum             # Modul dependensi Go
└── pkg/
    ├── blade/                  # Template engine Blade di atas Go
    ├── config/                 # Parser konfigurasi .env
    ├── db/                     # Abstraksi koneksi database (MySQL & SQLite)
    ├── engine/                 # Core AST Execution Engine & Scope Management
    ├── orm/                    # Emulasi Eloquent ORM & Query Executor
    ├── parser/                 # Lexer dan Parser sintaks PHP
    ├── router/                 # Dynamic HTTP Router & URL Pattern Matcher
    ├── security/               # HTTP security middleware & guard
    ├── slots/                  # Fitur bawaan Laravel (e.g. Sanctum Token)
    └── vendors/                # Virtual Vendor Registry & Provider slots
```

---

## 🚀 Persyaratan Sistem

- **Go**: Versi 1.22 atau yang lebih baru (direkomendasikan Go 1.24+)
- **Database (Opsional)**: MySQL / MariaDB atau SQLite

---

## 🛠️ Instalasi & Menjalankan

### 1. Clone Repositori
```bash
git clone https://github.com/larazeno/larazeno.git
cd larazeno
```

### 2. Unduh Dependensi Go
```bash
go mod tidy
```

### 3. Jalankan LaraZeno Menghadap ke Proyek Laravel
Arahkan flag `-path` ke folder proyek Laravel yang ingin dijalankan:

```bash
# Menjalankan pada direktori proyek Laravel tertentu
go run main.go -path=/path/ke/proyek/laravel -port=9001

# Atau jika dijalankan langsung di dalam folder proyek Laravel
go run /path/ke/larazeno/main.go -port=9001
```

Opsi argumen CLI:
- `-path` : Path absolut atau relatif ke direktori proyek Laravel (default: current working directory).
- `-port` : Port HTTP server yang akan dibuka (default: `9001`).

---

## 🗄️ CLI Migrasi Database (Zero PHP Artisan)

LaraZeno dilengkapi dengan built-in AST migration engine untuk membaca file-file migrasi di `database/migrations/` dan mengeksekusi skema tabel secara otomatis ke SQLite atau MySQL tanpa memerlukan PHP CLI atau Artisan:

```bash
# Menjalankan seluruh file migrasi yang pending
go run main.go -path=examples/laravel-official migrate

# Memeriksa status file migrasi (Pending vs Ran)
go run main.go -path=examples/laravel-official migrate:status
```

Fitur migrasi LaraZeno:
- Otomatis membuat dan mengelola tabel riwayat `migrations` (batching).
- Mendukung Schema Builder: `id()`, `string()`, `integer()`, `text()`, `timestamp()`, `timestamps()`, `rememberToken()`, `morphs()`.
- Mendukung modifiers: `->primary()`, `->unique()`, `->nullable()`.
- Idempoten: Migrasi yang sudah dieksekusi tidak akan dijalankan ulang.


---

## 📦 Menjalankan Real-World Example: ZenoPOS (Point of Sale & Inventory)

Repositori ini menyertakan contoh aplikasi kasir & manajemen inventaris (*Point of Sale*) di folder `examples/laravel-official`. Aplikasi ini menguji fitur Laravel 13 yang berjalan di atas LaraZeno: **Blade UI**, **Eloquent ORM**, **REST API**, dan **Database Migrations**.

### 1. Migrasi Database (Zero PHP)
Jalankan migrasi skema tabel `products`, `orders`, `users`, dan `personal_access_tokens`:

```bash
go run main.go -path=examples/laravel-official migrate
```

### 2. Menjalankan Server LaraZeno
```bash
go run main.go -path=examples/laravel-official -port=9006
```

LaraZeno akan secara otomatis menghubungkan database SQLite, mem-parsing controller `PosController` & `ProductController`, serta mendaftarkan route web & API.

### 3. Mengakses Antarmuka Aplikasi
Buka browser dan kunjungi:
- 🏠 **Halaman Beranda Default Laravel**: [http://localhost:9006/](http://localhost:9006/)
  - Menampilkan tampilan standar *welcome* Laravel 13 dengan navigasi langsung ke terminal POS dan manajemen produk.
- ⚡ **ZenoPOS Cashier Terminal**: [http://localhost:9006/pos](http://localhost:9006/pos)
  - **Katalog Produk Dinamis**: Pencarian instan dan filter kategori.
  - **Keranjang Kasir (Cart)**: Tambah kuantitas, hitung subtotal & total otomatis.
  - **Pilihan Pembayaran**: Tunai, QRIS, Kartu Debit.
  - **Struk Transaksi**: Cetak struk belanja (*Receipt dialog*) dan simpan riwayat order ke tabel `orders`.
- 📦 **Manajemen Inventaris Produk**: [http://localhost:9006/web/products](http://localhost:9006/web/products)
  - Operasi CRUD: Tambah produk, Edit cepat dengan modal dialog, dan Hapus produk.

### 4. Menguji REST API Endpoint (cURL)
```bash
# 1. Mendapatkan daftar produk
curl http://localhost:9006/api/products

# 2. Transaksi Checkout POS baru (Order)
curl -X POST http://localhost:9006/api/pos/checkout \
  -H "Content-Type: application/json" \
  -d '{"order_number": "ORD-88912", "customer_name": "Budi Santoso", "total_amount": 21000000, "payment_method": "QRIS", "items_json": "[{\"id\":\"2\",\"name\":\"MacBook Air M3\",\"qty\":1,\"price\":21000000}]"}'

# 3. Mendapatkan riwayat transaksi pesanan
curl http://localhost:9006/api/pos/orders
```


---

## 🌐 Contoh Penggunaan REST API

Ketika LaraZeno berjalan, rute yang didefinisikan pada folder `routes/` proyek Laravel Anda akan otomatis didaftarkan:

```bash
# Contoh endpoint publik
curl http://localhost:9001/api/products

# Contoh membuat data baru (POST)
curl -X POST http://localhost:9001/api/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Laptop Gaming", "price": 15000000}'

# Contoh endpoint yang diproteksi Sanctum
curl http://localhost:9001/api/user \
  -H "Authorization: Bearer <sanctum_access_token>"
```

---

## 🗺️ Roadmap Pengembangan

- [ ] Dukungan query builder lanjutan (`where`, `orderBy`, `paginate`).
- [ ] Relasi Eloquent (`hasMany`, `belongsTo`, `with`).
- [ ] Validasi Form Request komprehensif (`$request->validate(...)`).
- [ ] Event, Listener, dan Queue Dispatcher sederhana.

---

## 📄 Lisensi

Proyek ini didistribusikan di bawah lisensi [Apache 2.0](LICENSE).
