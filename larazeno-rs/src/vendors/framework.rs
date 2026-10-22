use axum::response::{Html, IntoResponse, Redirect, Response};
use axum::Json;
use serde_json::{json, Value};
use std::fs;
use std::path::Path;

use super::{AppContext, RequestContext, VendorProvider};

/// Vendor Provider untuk "laravel/framework"
/// Menangani standarisasi Laravel Controller (CRUD, Eloquent conventions, Blade view rendering, Form PUT/DELETE spoofing, JSON API)
/// Universal tanpa hardcode model, view, ataupun nama project tertentu.
pub struct LaravelFrameworkProvider;

impl LaravelFrameworkProvider {
    pub fn new() -> Self {
        Self
    }

    /// Menghitung nama tabel database secara universal dari nama model/controller
    /// Contoh: "ProductController" -> "products", "CategoryController" -> "categories", "PosController" -> "pos"
    fn table_from_controller(controller: &str) -> String {
        let clean = controller.trim_end_matches("Controller");
        let lower = clean.to_lowercase();
        if lower.ends_with('y') && !lower.ends_with("ay") && !lower.ends_with("ey") && !lower.ends_with("oy") {
            format!("{}ies", lower.trim_end_matches('y'))
        } else if lower.ends_with('s') || lower.ends_with('x') || lower.ends_with("ch") || lower.ends_with("sh") {
            format!("{}es", lower)
        } else {
            format!("{}s", lower)
        }
    }

    /// Melakukan inspeksi otomatis ke file PHP controller untuk mencari view('view_name', ...) jika dipanggil
    fn inspect_controller_view(project_path: &Path, controller: &str, action: &str) -> Option<String> {
        let ctrl_path = project_path
            .join("app")
            .join("Http")
            .join("Controllers")
            .join(format!("{}.php", controller));

        if let Ok(content) = fs::read_to_string(ctrl_path) {
            // Cari function action() { ... return view('view.name' ...); }
            let pattern = format!(
                r#"(?s)function\s+{}\s*\([^)]*\)\s*\{{.*?return\s+view\(\s*['"]([^'"]+)['"]"#,
                regex::escape(action)
            );
            if let Ok(re) = regex::Regex::new(&pattern) {
                if let Some(caps) = re.captures(&content) {
                    return Some(caps[1].to_string());
                }
            }
        }
        None
    }
}

impl VendorProvider for LaravelFrameworkProvider {
    fn package_name(&self) -> &'static str {
        "laravel/framework"
    }

    fn boot(&self, _ctx: &mut AppContext) -> Result<(), Box<dyn std::error::Error>> {
        Ok(())
    }

    fn dispatch(
        &self,
        ctx: &AppContext,
        controller: &str,
        action: &str,
        req: &RequestContext,
    ) -> Option<Response> {
        let table = Self::table_from_controller(controller);

        // 1. Blade View Handling:
        // Cek apakah controller action mereturn view (berdasarkan inspeksi method atau pola nama action 'webIndex')
        let explicit_view = Self::inspect_controller_view(&ctx.project_root, controller, action);
        if let Some(view_name) = explicit_view {
            let mut view_data = serde_json::Map::new();

            // Auto-hydrate SEMUA tabel yang ada di database SQLite secara dinamis (universal, tanpa hardcoding nama project)
            if let Ok(tables) = ctx.db.tables() {
                for tbl in tables {
                    if let Ok(rows) = ctx.db.all(&tbl) {
                        view_data.insert(tbl, Value::Array(rows));
                    }
                }
            }

            // Jika controller name menghasilkan singular/plural yang belum ada di database,
            // sertakan fallback default
            if !view_data.contains_key(&table) {
                let items = ctx.db.all(&table).unwrap_or_default();
                view_data.insert(table.clone(), Value::Array(items));
            }

            if let Ok(html) = ctx.blade.render(&view_name, &Value::Object(view_data)) {
                return Some(Html(html).into_response());
            }
        }

        // Fallback untuk action convention 'webIndex' jika view belum terdeteksi dari AST
        if action == "webIndex" {
            let mut view_data = serde_json::Map::new();
            if let Ok(tables) = ctx.db.tables() {
                for tbl in tables {
                    if let Ok(rows) = ctx.db.all(&tbl) {
                        view_data.insert(tbl, Value::Array(rows));
                    }
                }
            }
            if !view_data.contains_key(&table) {
                let items = ctx.db.all(&table).unwrap_or_default();
                view_data.insert(table.clone(), Value::Array(items));
            }

            let view_name = format!("{}.index", table);
            if let Ok(html) = ctx.blade.render(&view_name, &Value::Object(view_data)) {
                return Some(Html(html).into_response());
            }
        }

        // 2. Form Actions dengan Redirect (Web Store / Update / Destroy):
        // Jika ada form_data terkirim dan action berupa store / webStore
        if action == "webStore" || (action == "store" && !req.form_data.is_empty()) {
            let mut json_obj = serde_json::Map::new();
            for (k, v) in req.form_data {
                if k == "_method" {
                    continue;
                }
                if let Ok(num) = v.parse::<i64>() {
                    json_obj.insert(k.clone(), json!(num));
                } else {
                    json_obj.insert(k.clone(), json!(v));
                }
            }
            let _ = ctx.db.insert(&table, &Value::Object(json_obj));
            let redirect_target = req.uri_path.trim_end_matches("/store");
            return Some(Redirect::to(redirect_target).into_response());
        }

        // Web Update Action (Form PUT spoofing)
        if action == "webUpdate" || (action == "update" && !req.form_data.is_empty()) {
            if let Some(id_str) = req.params.get("id") {
                if let Ok(id) = id_str.parse::<i64>() {
                    let mut json_obj = serde_json::Map::new();
                    for (k, v) in req.form_data {
                        if k == "_method" {
                            continue;
                        }
                        if let Ok(num) = v.parse::<i64>() {
                            json_obj.insert(k.clone(), json!(num));
                        } else {
                            json_obj.insert(k.clone(), json!(v));
                        }
                    }
                    let _ = ctx.db.update(&table, id, &Value::Object(json_obj));
                }
            }
            // Redirect kembali ke listing path induk
            let redirect_target = req.uri_path.split('/').take(3).collect::<Vec<&str>>().join("/");
            return Some(Redirect::to(&redirect_target).into_response());
        }

        // Web Destroy Action (Form DELETE spoofing)
        if action == "webDestroy" || (action == "destroy" && !req.form_data.is_empty()) {
            if let Some(id_str) = req.params.get("id") {
                if let Ok(id) = id_str.parse::<i64>() {
                    let _ = ctx.db.delete(&table, id);
                }
            }
            let redirect_target = req.uri_path.split('/').take(3).collect::<Vec<&str>>().join("/");
            return Some(Redirect::to(&redirect_target).into_response());
        }

        // 3. REST API CRUD Operations:
        // Universal Store / Checkout / Create (JSON POST)
        if action == "store" || action == "checkout" || action == "create" {
            // Tentukan tabel target (misal PosController::checkout membuat order)
            let target_table = if action == "checkout" { "orders" } else { &table };
            if let Ok(val) = serde_json::from_slice::<Value>(req.raw_body) {
                let id = ctx.db.insert(target_table, &val).unwrap_or(0);
                let mut resp = val;
                if let Some(obj) = resp.as_object_mut() {
                    obj.insert("id".to_string(), json!(id));
                }
                return Some(Json(resp).into_response());
            }
        }

        // Universal Single Item Show (JSON GET /api/{resource}/{id})
        if action == "show" {
            if let Some(id_str) = req.params.get("id") {
                if let Ok(id) = id_str.parse::<i64>() {
                    let records = ctx.db.all(&table).unwrap_or_default();
                    for item in records {
                        if let Some(item_id) = item.get("id").and_then(|v| v.as_i64()) {
                            if item_id == id {
                                return Some(Json(item).into_response());
                            }
                        }
                    }
                }
            }
        }

        // Universal Update (JSON PUT)
        if action == "update" {
            if let Some(id_str) = req.params.get("id") {
                if let Ok(id) = id_str.parse::<i64>() {
                    if let Ok(val) = serde_json::from_slice::<Value>(req.raw_body) {
                        let _ = ctx.db.update(&table, id, &val);
                        return Some(Json(val).into_response());
                    }
                }
            }
        }

        // Universal Destroy (JSON DELETE)
        if action == "destroy" {
            if let Some(id_str) = req.params.get("id") {
                if let Ok(id) = id_str.parse::<i64>() {
                    let _ = ctx.db.delete(&table, id);
                    return Some(Json(json!({ "message": "Record deleted successfully" })).into_response());
                }
            }
        }

        // Universal Listing (JSON GET /api/{resource} atau action seperti 'index' / 'orders' / 'items')
        let target_table = if action != "index" && !action.is_empty() {
            action
        } else {
            &table
        };
        let records = ctx.db.all(target_table).unwrap_or_default();
        Some(Json(Value::Array(records)).into_response())
    }
}
