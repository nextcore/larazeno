# LaraZeno
 
**LaraZeno** adalah *high-performance Laravel execution engine & runtime* yang dirancang untuk mengeksekusi kode sumber Laravel langsung melalui **Abstract Syntax Tree (AST)** tanpa memerlukan instalasi runtime PHP, Composer, maupun folder fisik `vendor/` di lingkungan host (*Zero PHP Runtime, Zero Composer, Zero Vendor Folder*).

LaraZeno kini hadir dalam dua implementasi native:
- **Go Engine (`larazeno`)**: Fleksibel, ringan, dan kaya fitur virtual vendor loader.
- **Rust Engine (`larazeno-rs`)**: Berbasis **Rust 1.99+ (Edition 2024)**, **Axum**, dan **Tokio** untuk throughput tinggi dengan latensi ultra-rendah.

---

## ⚡ Fitur Utama

- **Zero Composer & Zero Vendor Folder**: Tidak perlu `composer install` atau memelihara folder `vendor` yang memakan ruang ratusan megabyte.
- **Universal Multi-Runtime**: Tersedia versi **Go** dan **Rust** dengan binary mandiri (*single static binary*).
- **PHP Lexer & AST Parser**: Mengurai sintaks controller, routes, dan model Laravel langsung ke representasi AST di memori.
- **Universal REST API Engine**:
  - Auto-discovery route (`routes/api.php`, `routes/web.php`, dll).
  - Mendukung `Route::get`, `Route::post`, `Route::put`, `Route::delete`.
  - Mendukung `Route::apiResource('endpoint', Controller::class)`.
  - Penanganan parameter dinamis URL (e.g. `/{id}`).
- **Built-in Authentication (Laravel Sanctum)**:
  - Dukungan middleware `auth:sanctum`.
  - Validasi *Bearer Token* dan resolusi `$request->user()`.
- **Lightweight Eloquent ORM Emulation**:
  - Operasi dasar: `Model::all()`, `Model::find($id)`, `Model::create($data)`, `$model->update()`, `$model->delete()`.
  - Mendukung driver database **MySQL / MariaDB** dan **SQLite** sesuai konfigurasi `.env`.
- **Blade Template Engine (Pure AST)**:
  - Menggunakan **Pure Recursive Descent AST Lexer, Parser & Evaluator** (Zero Regex) untuk Blade template.
  - Mendukung direktif `@if`, `@else`, `@elseif`, `@endif`, `@foreach`, `@endforeach`, `@csrf`, `@method`, `@include`, interpolasi variabel `{{ $var }}`, property access `{{ $product.price }}` / `{{ $product['price'] }}`, dan helper functions `config()`, `app()->version()`.
  - **Dynamic Schema Auto-Hydration**: Data view otomatis terhidrasi dari tabel database SQLite/MySQL tanpa hardcoding.
- **Built-in Migration CLI**:
  - Migrasi database otomatis (`migrate` dan `migrate:status`) membaca file `database/migrations/` tanpa PHP Artisan.
- **Security Guard**:
  - Proteksi terintegrasi dari akses file `.env`, webshell, atau eksekusi skrip ilegal langsung.

---

## 📁 Struktur Direktori

```text
larazeno/
├── larazeno-go/                 # Engine LaraZeno berbasis Go
│   ├── main.go                  # Entry point CLI, Migration & HTTP Dispatcher
│   ├── go.mod / go.sum          # Modul dependensi Go
│   └── pkg/
│       ├── blade/               # Template engine Blade di atas Go
│       ├── config/              # Parser konfigurasi .env
│       ├── db/                  # Abstraksi koneksi database (MySQL & SQLite)
│       ├── engine/              # Core AST Execution Engine & Scope Management
│       ├── orm/                 # Emulasi Eloquent ORM & Query Executor
│       ├── parser/              # Lexer dan Parser sintaks PHP
│       ├── router/              # Dynamic HTTP Router & URL Pattern Matcher
│       ├── security/            # HTTP security middleware & guard
│       └── vendors/             # Virtual Vendor Registry & Provider slots
├── larazeno-rs/                 # Engine LaraZeno berbasis Rust (Edition 2024, Axum, Tokio)
│   ├── Cargo.toml               # Dependensi crate Rust
│   └── src/
│       ├── blade.rs             # Pure Recursive Descent AST Blade Lexer, Parser & Evaluator
│       ├── cli.rs               # Universal Migration CLI (migrate & migrate:status)
│       ├── db.rs                # SQLite thread-safe pool & dynamic table schema auto-discovery
│       ├── main.rs              # Axum HTTP Server & Dynamic Route Matcher
│       ├── parser.rs            # PHP AST Parser berbasis tree-sitter-php
│       └── vendors/             # Modular Virtual Vendor System (Trait-based Provider)
└── examples/
    └── laravel-official/        # Real-world project Laravel 13 (ZenoPOS, Blade & Sanctum)
```

---

## 🚀 Persyaratan Sistem

- **Go**: Versi 1.22 atau yang lebih baru (direkomendasikan Go 1.24+) untuk `larazeno-go`
- **Rust**: Versi 1.85+ / 1.99+ (Rust Edition 2024) dan `cargo` untuk `larazeno-rs`
- **Database (Opsional)**: SQLite (bawaan) atau MySQL / MariaDB

---

## 🛠️ Instalasi & Menjalankan

### 1. Clone Repositori
```bash
git clone https://github.com/larazeno/larazeno.git
cd larazeno
```

### 2. Menjalankan LaraZeno Go (`larazeno-go`)
```bash
cd larazeno-go
go mod tidy

# Menjalankan server mengarah ke project Laravel target
go run main.go -path=../examples/laravel-official -port=9006
```

### 3. Menjalankan LaraZeno Rust (`larazeno-rs`)
```bash
cd larazeno-rs

# Menjalankan server performa tinggi Axum mengarah ke project Laravel target
cargo run -- --path=../examples/laravel-official --port=9008
```

Opsi argumen CLI:
- `--path` / `-path` : Path ke direktori proyek Laravel (default: current working directory).
- `--port` / `-port` : Port HTTP server yang akan dibuka (default: `9008` di Rust, `9001` di Go).

---

## 🗄️ CLI Migrasi Database (Zero PHP Artisan)

LaraZeno dilengkapi dengan built-in AST migration engine di **Go** dan **Rust** untuk membaca file-file migrasi di `database/migrations/` dan mengeksekusi skema tabel secara otomatis ke SQLite atau MySQL tanpa memerlukan PHP CLI atau Artisan:

```bash
# Opsi A: Menggunakan Go CLI (larazeno-go)
cd larazeno-go
go run main.go -path=../examples/laravel-official migrate
go run main.go -path=../examples/laravel-official migrate:status

# Opsi B: Menggunakan Rust CLI (larazeno-rs)
cd larazeno-rs
cargo run -- --path=../examples/laravel-official migrate
cargo run -- --path=../examples/laravel-official migrate:status
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
# Opsi A: Menggunakan Go CLI (larazeno-go)
cd larazeno-go
go run main.go -path=../examples/laravel-official migrate
go run main.go -path=../examples/laravel-official migrate:status

# Opsi B: Menggunakan Rust CLI (larazeno-rs)
cd larazeno-rs
cargo run -- --path=../examples/laravel-official migrate
cargo run -- --path=../examples/laravel-official migrate:status
```

### 2. Menjalankan Server LaraZeno
Anda dapat memilih untuk menjalankan engine berbasis **Go** atau **Rust**:

```bash
# Opsi A: Menggunakan Go Engine (Port 9006)
cd larazeno-go
go run main.go -path=../examples/laravel-official -port=9006

# Opsi B: Menggunakan Rust Engine (Port 9008)
cd larazeno-rs
cargo run -- --path=../examples/laravel-official --port=9008
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

## 🧩 Modular Virtual Vendor Architecture (Go & Rust)

LaraZeno mengusung arsitektur **Virtual Vendor Provider** yang sepenuhnya modular. Vendor PHP Laravel tidak dieksekusi secara lambat, melainkan digantikan oleh provider native (Go / Rust) dengan kontrak package composer yang identik:

### 1. Bagaimana Menambahkan Vendor Baru di Rust (`larazeno-rs`)

Cukup buat file di [src/vendors/](file:///media/max/3C38C268119D6B5C/github/larazeno/larazeno-rs/src/vendors) dan implementasikan trait [VendorProvider](file:///media/max/3C38C268119D6B5C/github/larazeno/larazeno-rs/src/vendors/mod.rs):

```rust
use axum::response::Response;
use crate::vendors::{AppContext, RequestContext, VendorProvider};

pub struct MyCustomVendor;

impl VendorProvider for MyCustomVendor {
    fn package_name(&self) -> &'static str {
        "acme/custom-package"
    }

    fn boot(&self, ctx: &mut AppContext) -> Result<(), Box<dyn std::error::Error>> {
        // Inisialisasi plugin/migrasi/state
        Ok(())
    }

    fn dispatch(
        &self,
        ctx: &AppContext,
        controller: &str,
        action: &str,
        req: &RequestContext,
    ) -> Option<Response> {
        if controller == "AcmeController" {
            // Tangani request dan kembalikan axum Response
            return Some(axum::Json(serde_json::json!({"status": "ok"})).into_response());
        }
        None // Lempar ke vendor berikutnya jika tidak cocok
    }
}
```

Daftarkan di `main.rs`:
```rust
registry.register(Box::new(MyCustomVendor));
```
Vendor hanya akan aktif jika dicantumkan di bagian `"require"` pada `composer.json` project target!

---

## 🗺️ Roadmap Pengembangan

- [x] Trait-based Virtual Vendor Provider modular (`larazeno-rs`).
- [x] Pure AST Lexer, Parser, & Evaluator untuk Blade Template Engine (Zero Regex).
- [ ] Dukungan query builder lanjutan (`where`, `orderBy`, `paginate`).
- [ ] Relasi Eloquent (`hasMany`, `belongsTo`, `with`).
- [ ] Validasi Form Request komprehensif (`$request->validate(...)`).
- [ ] Event, Listener, dan Queue Dispatcher sederhana.

---

## 📄 Lisensi

Proyek ini didistribusikan di bawah lisensi [Apache 2.0](LICENSE).
