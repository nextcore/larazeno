use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::Json;
use serde_json::json;

use super::{AppContext, RequestContext, VendorProvider};

/// Vendor Provider untuk "laravel/sanctum"
/// Menangani token generation dan auth guard endpoints ($request->user() resolution)
pub struct LaravelSanctumProvider;

impl LaravelSanctumProvider {
    pub fn new() -> Self {
        Self
    }
}

impl VendorProvider for LaravelSanctumProvider {
    fn package_name(&self) -> &'static str {
        "laravel/sanctum"
    }

    fn boot(&self, _ctx: &mut AppContext) -> Result<(), Box<dyn std::error::Error>> {
        Ok(())
    }

    fn dispatch(
        &self,
        ctx: &AppContext,
        _controller: &str,
        action: &str,
        _req: &RequestContext,
    ) -> Option<Response> {
        // Universal Sanctum action endpoints jika diarahkan via Controller (e.g. SanctumController / AuthController)
        if action == "login" || action == "createToken" {
            let token = format!("{}-{}", "1|sanctum_token", chrono_fake_or_random());
            return Some(
                Json(json!({
                    "token_type": "Bearer",
                    "access_token": token,
                    "message": "Authenticated via LaraZeno Sanctum Virtual Vendor"
                }))
                .into_response(),
            );
        } else if action == "user" {
            let users = ctx.db.all("users").unwrap_or_default();
            if let Some(first_user) = users.into_iter().next() {
                return Some(Json(first_user).into_response());
            } else {
                return Some(
                    Json(json!({
                        "id": 1,
                        "name": "LaraZeno User",
                        "email": "user@larazeno.dev"
                    }))
                    .into_response(),
                );
            }
        }

        None
    }

    fn handle_closure(
        &self,
        ctx: &AppContext,
        closure_type: &str,
        req: &RequestContext,
    ) -> Option<Response> {
        if closure_type == "auth_user" {
            // Cek Authorization Bearer Token
            let auth_header = req.headers.get("authorization").and_then(|h| h.to_str().ok());
            
            // Jika ada Bearer token atau pada mode dev/testing, resolve user pertama dari tabel users
            if let Some(hdr) = auth_header {
                if !hdr.to_lowercase().starts_with("bearer ") {
                    return Some((StatusCode::UNAUTHORIZED, "Invalid Bearer Token").into_response());
                }
            }

            let users = ctx.db.all("users").unwrap_or_default();
            if let Some(first_user) = users.into_iter().next() {
                return Some(Json(first_user).into_response());
            } else {
                return Some(
                    Json(json!({
                        "id": 1,
                        "name": "LaraZeno User",
                        "email": "user@larazeno.dev"
                    }))
                    .into_response(),
                );
            }
        }

        None
    }
}

fn chrono_fake_or_random() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    let d = SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default();
    format!("{:x}", d.as_nanos())
}
