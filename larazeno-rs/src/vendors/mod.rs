use axum::http::HeaderMap;
use axum::response::Response;
use serde_json::Value;
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;

use crate::blade::BladeEngine;
use crate::db::DbPool;
use crate::router::DynamicRoute;

/// AppContext menyimpan state runtime aplikasi yang dibagi dan dapat diakses oleh semua vendor
pub struct AppContext {
    pub project_root: PathBuf,
    pub db: DbPool,
    pub blade: Arc<BladeEngine>,
    pub routes: Vec<DynamicRoute>,
    pub packages: HashMap<String, String>, // misal: "laravel/framework" -> "^13.17"
    pub env: HashMap<String, String>,
}

impl AppContext {
    pub fn new(project_root: PathBuf, db: DbPool, blade: Arc<BladeEngine>) -> Self {
        Self {
            project_root,
            db,
            blade,
            routes: Vec::new(),
            packages: HashMap::new(),
            env: HashMap::new(),
        }
    }
}

/// Dynamic Request Context untuk diteruskan ke vendor controller hook
pub struct RequestContext<'a> {
    pub method: &'a str,
    pub uri_path: &'a str,
    pub headers: &'a HeaderMap,
    pub params: &'a HashMap<String, String>,
    pub raw_body: &'a [u8],
    pub form_data: &'a HashMap<String, String>,
}

/// VendorProvider adalah trait modular yang harus diimplementasikan oleh setiap Virtual Vendor di LaraZeno-rs
pub trait VendorProvider: Send + Sync {
    /// Nama package composer, misal: "laravel/framework", "laravel/sanctum", "inertiajs/inertia-laravel"
    fn package_name(&self) -> &'static str;

    /// Lifecycle hook: Dipanggil saat vendor di-load (misal menambah custom routes, migration hooks, dsb.)
    fn boot(&self, _ctx: &mut AppContext) -> Result<(), Box<dyn std::error::Error>> {
        Ok(())
    }

    /// Dispatcher hook: Apakah vendor ini menangani controller / action ini?
    /// Mengembalikan Some(Response) jika ditangani, atau None untuk dilempar ke vendor berikutnya / fallback default.
    fn dispatch(
        &self,
        _ctx: &AppContext,
        _controller: &str,
        _action: &str,
        _req: &RequestContext,
    ) -> Option<Response> {
        None
    }

    /// Closure hook: Apakah vendor ini menangani closure khusus seperti auth $request->user()?
    fn handle_closure(
        &self,
        _ctx: &AppContext,
        _closure_type: &str,
        _req: &RequestContext,
    ) -> Option<Response> {
        None
    }
}

/// Registry mengelola semua VendorProvider yang terdaftar
pub struct VendorRegistry {
    providers: HashMap<&'static str, Box<dyn VendorProvider>>,
    active_vendor_keys: Vec<&'static str>,
}

impl VendorRegistry {
    pub fn new() -> Self {
        Self {
            providers: HashMap::new(),
            active_vendor_keys: Vec::new(),
        }
    }

    /// Mendaftarkan implementasi vendor provider
    pub fn register(&mut self, provider: Box<dyn VendorProvider>) {
        self.providers.insert(provider.package_name(), provider);
    }

    /// Membaca composer.json di target project dan mengaktifkan vendor-vendor yang sesuai
    pub fn load_from_composer(&mut self, ctx: &mut AppContext) -> Result<(), Box<dyn std::error::Error>> {
        let composer_path = ctx.project_root.join("composer.json");
        if !composer_path.exists() {
            println!("   ⚠️  composer.json tidak ditemukan di {:?}", ctx.project_root);
            return Ok(());
        }

        let content = std::fs::read_to_string(&composer_path)?;
        let parsed: Value = serde_json::from_str(&content)?;

        let mut required_packages = HashMap::new();
        if let Some(req_obj) = parsed.get("require").and_then(|r| r.as_object()) {
            for (pkg, ver) in req_obj {
                if let Some(v_str) = ver.as_str() {
                    required_packages.insert(pkg.clone(), v_str.to_string());
                }
            }
        }

        ctx.packages = required_packages.clone();

        println!("📦 Mengaktifkan Virtual Vendors berdasarkan composer.json:");
        for (pkg_name, provider) in &self.providers {
            if required_packages.contains_key(*pkg_name) {
                println!(
                    "   [vendor: active] 🔌 {} ({})",
                    pkg_name,
                    required_packages.get(*pkg_name).unwrap_or(&"latest".to_string())
                );
                self.active_vendor_keys.push(*pkg_name);
                provider.boot(ctx)?;
            }
        }

        Ok(())
    }

    /// Dispatch eksekusi controller ke vendor-vendor aktif secara berurutan
    pub fn dispatch(
        &self,
        ctx: &AppContext,
        controller: &str,
        action: &str,
        req: &RequestContext,
    ) -> Option<Response> {
        for key in &self.active_vendor_keys {
            if let Some(provider) = self.providers.get(key) {
                if let Some(res) = provider.dispatch(ctx, controller, action, req) {
                    return Some(res);
                }
            }
        }
        None
    }

    /// Dispatch penanganan closure route khusus ke vendor aktif
    pub fn handle_closure(
        &self,
        ctx: &AppContext,
        closure_type: &str,
        req: &RequestContext,
    ) -> Option<Response> {
        for key in &self.active_vendor_keys {
            if let Some(provider) = self.providers.get(key) {
                if let Some(res) = provider.handle_closure(ctx, closure_type, req) {
                    return Some(res);
                }
            }
        }
        None
    }
}

pub mod framework;
pub mod inertia;
pub mod sanctum;
