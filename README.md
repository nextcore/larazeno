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
