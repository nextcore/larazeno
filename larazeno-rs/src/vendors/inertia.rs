use axum::response::{Html, IntoResponse, Response};
use serde_json::json;
use std::fs;

use super::{AppContext, RequestContext, VendorProvider};

/// Vendor Provider untuk "inertiajs/inertia-laravel"
/// Menangani rendering Inertia HTML shell atau JSON protocol payload secara dinamis
pub struct InertiaProvider;

impl InertiaProvider {
    pub fn new() -> Self {
        Self
    }

    /// Deteksi apakah controller action mengembalikan Inertia::render('Component', ...)
    fn inspect_inertia_render(ctx: &AppContext, controller: &str, action: &str) -> Option<String> {
        let ctrl_path = ctx
            .project_root
            .join("app")
            .join("Http")
            .join("Controllers")
            .join(format!("{}.php", controller));

        if let Ok(content) = fs::read_to_string(ctrl_path) {
            let pattern = format!(
                r#"(?s)function\s+{}\s*\([^)]*\)\s*\{{.*?Inertia::render\(\s*['"]([^'"]+)['"]"#,
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

impl VendorProvider for InertiaProvider {
    fn package_name(&self) -> &'static str {
        "inertiajs/inertia-laravel"
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
        // Cek apakah controller action menggunakan Inertia
        let explicit_component = Self::inspect_inertia_render(ctx, controller, action);
        let is_inertia_convention = action.starts_with("inertia") || action.ends_with("Inertia");

        if explicit_component.is_some() || is_inertia_convention {
            let component = explicit_component.unwrap_or_else(|| "Dashboard".to_string());
            let props = json!({
                "auth": { "user": { "name": "LaraZeno Admin" } },
                "flash": {}
            });

            // Cek header X-Inertia
            let is_inertia_request = req
                .headers
                .get("x-inertia")
                .and_then(|h| h.to_str().ok())
                .map(|v| v.eq_ignore_ascii_case("true"))
                .unwrap_or(false);

            let page_data = json!({
                "component": component,
                "props": props,
                "url": req.uri_path,
                "version": "1.0"
            });

            if is_inertia_request {
                return Some(
                    axum::Json(page_data)
                        .into_response()
                );
            }

            // Browser Full Page Visit: Render app.blade.php shell
            let page_json_str = serde_json::to_string(&page_data).unwrap_or_default();
            let page_attr = page_json_str.replace('\'', "&#039;");

            let app_markup = format!(
                r#"<script data-page="app" type="application/json">{}</script><div id="app" data-page='{}'></div>"#,
                page_json_str, page_attr
            );

            // Render root app template via Blade jika ada
            if let Ok(mut html) = ctx.blade.render("app", &json!({})) {
                html = html.replace("@inertia", &app_markup);
                return Some(Html(html).into_response());
            }

            let fallback_html = format!(
                r#"<!DOCTYPE html>
<html>
<head><title>LaraZeno Inertia</title></head>
<body>
{}
</body>
</html>"#,
                app_markup
            );

            return Some(Html(fallback_html).into_response());
        }

        None
    }
}
