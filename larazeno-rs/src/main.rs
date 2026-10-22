mod blade;
mod db;
mod migrate;
mod router;
pub mod vendors;

use axum::{
    extract::{Request, State},
    http::{HeaderMap, StatusCode},
    response::{Html, IntoResponse, Redirect, Response},
    Router,
};
use clap::Parser;
use router::{RouteHandler, RouteScanner};
use serde_json::json;
use std::collections::HashMap;
use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::Arc;
use vendors::{AppContext, RequestContext, VendorRegistry};

#[derive(Parser, Debug)]
#[command(author, version, about = "LaraZeno-rs: Pure Rust High-Performance Laravel 13 Runtime")]
struct Args {
    #[arg(long, default_value = "examples/laravel-official")]
    path: String,

    #[arg(short, long, default_value = "9008")]
    port: u16,

    /// Subcommand: migrate or migrate:status
    command: Option<String>,
}

#[derive(Clone)]
struct AppState {
    ctx: Arc<AppContext>,
    registry: Arc<VendorRegistry>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse();
    let project_path = PathBuf::from(&args.path);

    println!("================================================================");
    println!("  LaraZeno-rs: High-Performance Laravel Runtime (Pure Rust)     ");
    println!("  Zero PHP | Zero Composer | Universal Modular Architecture     ");
    println!("================================================================");
    println!("📁 Project Root: {:?}", project_path);

    // Initialize Database (SQLite)
    let db_path = project_path.join("database").join("database.sqlite");
    let db_str = db_path.to_str().unwrap_or("database.sqlite");
    let pool = db::DbPool::new(db_str)?;
    println!("✔ Connected to SQLite: {}", db_str);

    // Subcommand: migrate or migrate:status
    if let Some(cmd) = args.command.as_deref() {
        if cmd == "migrate" {
            let runner = migrate::MigrationRunner::new(&project_path, pool);
            runner.run()?;
            return Ok(());
        } else if cmd == "migrate:status" {
            let runner = migrate::MigrationRunner::new(&project_path, pool);
            runner.status()?;
            return Ok(());
        } else {
            eprintln!(
                "Perintah tidak dikenal: '{}'. Gunakan 'migrate' atau 'migrate:status'",
                cmd
            );
            return Ok(());
        }
    }

    // Dynamic Route Auto-Discovery from routes/web.php & routes/api.php
    let scanner = RouteScanner::new(&project_path);
    let scanned_routes = scanner.scan_routes();
    println!("📦 Dynamic Route Auto-Discovery:");
    for r in &scanned_routes {
        println!(
            "   [laravel: Route] {:<6} {} -> {:?}",
            r.method, r.path_pattern, r.handler
        );
    }
    println!("✔ Total Routes Registered: {}", scanned_routes.len());

    let blade = Arc::new(blade::BladeEngine::new(&project_path));

    // Initialize AppContext
    let mut app_ctx = AppContext::new(project_path.clone(), pool, blade);
    app_ctx.routes = scanned_routes;

    // Initialize Modular Vendor Registry
    let mut registry = VendorRegistry::new();
    registry.register(Box::new(vendors::framework::LaravelFrameworkProvider::new()));
    registry.register(Box::new(vendors::sanctum::LaravelSanctumProvider::new()));
    registry.register(Box::new(vendors::inertia::InertiaProvider::new()));

    // Auto-load vendor plugins based on composer.json
    registry.load_from_composer(&mut app_ctx)?;

    let state = AppState {
        ctx: Arc::new(app_ctx),
        registry: Arc::new(registry),
    };

    // Universal Dynamic Catch-All Router
    let app = Router::new()
        .fallback(handle_dynamic_request)
        .with_state(state);

    let addr = SocketAddr::from(([127, 0, 0, 1], args.port));
    println!("🚀 LaraZeno-rs listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}

async fn handle_dynamic_request(State(state): State<AppState>, req: Request) -> Response {
    let raw_method = req.method().as_str().to_string();
    let path = req.uri().path().to_string();

    // Extract headers
    let headers = req.headers().clone();

    // Parse Body Bytes to handle Form Data / JSON and Method Spoofing (_method=PUT/DELETE)
    let (_parts, body) = req.into_parts();
    let bytes = match axum::body::to_bytes(body, 1024 * 1024 * 10).await {
        Ok(b) => b,
        Err(_) => return (StatusCode::BAD_REQUEST, "Payload too large").into_response(),
    };

    let mut form_data = HashMap::new();
    let mut method = raw_method.clone();

    // Check if form-urlencoded
    if let Ok(form_str) = std::str::from_utf8(&bytes) {
        if let Ok(parsed) = serde_urlencoded::from_str::<HashMap<String, String>>(form_str) {
            if let Some(m) = parsed.get("_method") {
                method = m.to_uppercase();
            }
            form_data = parsed;
        }
    }

    // Match against auto-discovered dynamic routes
    for r in state.ctx.routes.iter() {
        if r.method != method {
            continue;
        }

        if let Some(caps) = r.regex.captures(&path) {
            let mut params = HashMap::new();
            for (idx, name) in r.param_names.iter().enumerate() {
                if let Some(val) = caps.get(idx + 1) {
                    params.insert(name.clone(), val.as_str().to_string());
                }
            }

            return dispatch_route_handler(
                &state,
                &r.handler,
                &method,
                &path,
                &headers,
                &params,
                &bytes,
                &form_data,
            )
            .await;
        }
    }

    (
        StatusCode::NOT_FOUND,
        format!("404 Not Found: {} {}", method, path),
    )
        .into_response()
}

async fn dispatch_route_handler(
    state: &AppState,
    handler: &RouteHandler,
    method: &str,
    uri_path: &str,
    headers: &HeaderMap,
    params: &HashMap<String, String>,
    raw_body: &[u8],
    form_data: &HashMap<String, String>,
) -> Response {
    let req_ctx = RequestContext {
        method,
        uri_path,
        headers,
        params,
        raw_body,
        form_data,
    };

    match handler {
        RouteHandler::ClosureView(view_name) => {
            match state.ctx.blade.render(view_name, &json!({})) {
                Ok(html) => Html(html).into_response(),
                Err(err) => (StatusCode::INTERNAL_SERVER_ERROR, err).into_response(),
            }
        }
        RouteHandler::ClosureRedirect(target_url) => Redirect::to(target_url).into_response(),
        RouteHandler::ClosureAuthUser => {
            // Forward to sanctum / auth vendor handler
            if let Some(resp) = state.registry.handle_closure(&state.ctx, "auth_user", &req_ctx) {
                return resp;
            }
            (
                StatusCode::UNAUTHORIZED,
                "Authentication provider not loaded or user not authenticated",
            )
                .into_response()
        }
        RouteHandler::ControllerAction { controller, action } => {
            // Delegate controller dispatching through modular Virtual Vendors
            if let Some(resp) = state.registry.dispatch(&state.ctx, controller, action, &req_ctx) {
                return resp;
            }

            (
                StatusCode::NOT_FOUND,
                format!("Unhandled Controller Action: {}@{}", controller, action),
            )
                .into_response()
        }
    }
}
