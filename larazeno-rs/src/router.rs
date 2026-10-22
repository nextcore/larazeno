use regex::Regex;
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone)]
pub enum RouteHandler {
    ClosureView(String),    // e.g. return view('welcome')
    ClosureRedirect(String),// e.g. return redirect('/pos')
    ClosureAuthUser,        // e.g. return $request->user() with auth:sanctum
    ControllerAction {
        controller: String, // e.g. "ProductController"
        action: String,     // e.g. "webIndex"
    },
}

#[derive(Debug, Clone)]
pub struct DynamicRoute {
    pub method: String,
    pub path_pattern: String,
    pub regex: Regex,
    pub param_names: Vec<String>,
    pub handler: RouteHandler,
}

pub struct RouteScanner {
    project_path: PathBuf,
}

impl RouteScanner {
    pub fn new(project_path: &Path) -> Self {
        Self {
            project_path: project_path.to_path_buf(),
        }
    }

    pub fn scan_routes(&self) -> Vec<DynamicRoute> {
        let mut routes = Vec::new();
        let routes_dir = self.project_path.join("routes");

        let web_file = routes_dir.join("web.php");
        if web_file.exists() {
            if let Ok(content) = fs::read_to_string(&web_file) {
                routes.extend(self.parse_php_routes(&content, ""));
            }
        }

        let api_file = routes_dir.join("api.php");
        if api_file.exists() {
            if let Ok(content) = fs::read_to_string(&api_file) {
                // Laravel standards: routes defined in api.php automatically get /api prefix
                routes.extend(self.parse_php_routes(&content, "/api"));
            }
        }

        routes
    }

    fn parse_php_routes(&self, content: &str, prefix: &str) -> Vec<DynamicRoute> {
        let mut routes = Vec::new();

        // 1. Parse Route::apiResource('products', Controller::class)
        let re_resource = Regex::new(
            r#"Route::apiResource\s*\(\s*['"]([^'"]+)['"]\s*,\s*\\?([a-zA-Z0-9_\\]+)::class"#
        ).unwrap();

        for caps in re_resource.captures_iter(content) {
            let res_name = &caps[1];
            let raw_ctrl = &caps[2];
            let ctrl = raw_ctrl.split('\\').last().unwrap_or(raw_ctrl).to_string();

            let base_path = format!("{}/{}", prefix, res_name.trim_matches('/'));
            let clean_base = if base_path.starts_with('/') { base_path } else { format!("/{}", base_path) };

            routes.push(self.create_route("GET", &clean_base, RouteHandler::ControllerAction {
                controller: ctrl.clone(),
                action: "index".to_string(),
            }));
            routes.push(self.create_route("POST", &clean_base, RouteHandler::ControllerAction {
                controller: ctrl.clone(),
                action: "store".to_string(),
            }));
            routes.push(self.create_route("GET", &format!("{}/{{id}}", clean_base), RouteHandler::ControllerAction {
                controller: ctrl.clone(),
                action: "show".to_string(),
            }));
            routes.push(self.create_route("PUT", &format!("{}/{{id}}", clean_base), RouteHandler::ControllerAction {
                controller: ctrl.clone(),
                action: "update".to_string(),
            }));
            routes.push(self.create_route("DELETE", &format!("{}/{{id}}", clean_base), RouteHandler::ControllerAction {
                controller: ctrl,
                action: "destroy".to_string(),
            }));
        }

        // 2. Parse Route::get/post/put/delete with Controller action: [ProductController::class, 'webIndex']
        let re_controller_routes = Regex::new(
            r#"Route::(get|post|put|delete|patch)\s*\(\s*['"]([^'"]+)['"]\s*,\s*\[\s*([a-zA-Z0-9_\\]+)::class\s*,\s*['"]([^'"]+)['"]\s*\]\s*\)"#
        ).unwrap();

        for caps in re_controller_routes.captures_iter(content) {
            let method = caps[1].to_uppercase();
            let path = &caps[2];
            let raw_ctrl = &caps[3];
            let ctrl = raw_ctrl.split('\\').last().unwrap_or(raw_ctrl).to_string();
            let action = caps[4].to_string();

            let full_path = format!("{}/{}", prefix, path.trim_matches('/'));
            let clean_path = if full_path == "/" || full_path.starts_with('/') {
                if full_path.len() > 1 && full_path.ends_with('/') {
                    full_path.trim_end_matches('/').to_string()
                } else {
                    full_path
                }
            } else {
                format!("/{}", full_path)
            };

            routes.push(self.create_route(&method, &clean_path, RouteHandler::ControllerAction {
                controller: ctrl,
                action,
            }));
        }

        // 3. Parse Route::get('/user', function (Request $request) { return $request->user(); })
        let re_auth_user = Regex::new(
            r#"Route::(get|post)\s*\(\s*['"]([^'"]+)['"]\s*,\s*function\s*\([^)]*\)\s*\{[^}]*\$request->user\(\)[^}]*\}\s*\)"#
        ).unwrap();

        for caps in re_auth_user.captures_iter(content) {
            let method = caps[1].to_uppercase();
            let path = &caps[2];
            let full_path = format!("{}/{}", prefix, path.trim_matches('/'));
            let clean_path = if full_path.starts_with('/') { full_path } else { format!("/{}", full_path) };

            routes.push(self.create_route(&method, &clean_path, RouteHandler::ClosureAuthUser));
        }

        // 4. Parse Route::get('/', function () { return view('welcome'); })
        let re_view_closure = Regex::new(
            r#"Route::(get|post)\s*\(\s*['"]([^'"]+)['"]\s*,\s*function\s*\([^)]*\)\s*\{[^}]*return\s+view\(\s*['"]([^'"]+)['"]\s*\)[^}]*\}\s*\)"#
        ).unwrap();

        for caps in re_view_closure.captures_iter(content) {
            let method = caps[1].to_uppercase();
            let path = &caps[2];
            let view_name = caps[3].to_string();

            let full_path = format!("{}/{}", prefix, path.trim_matches('/'));
            let clean_path = if full_path.starts_with('/') { full_path } else { format!("/{}", full_path) };

            routes.push(self.create_route(&method, &clean_path, RouteHandler::ClosureView(view_name)));
        }

        // 5. Parse Route::get('/', function () { return redirect('/pos'); })
        let re_redirect_closure = Regex::new(
            r#"Route::(get|post)\s*\(\s*['"]([^'"]+)['"]\s*,\s*function\s*\([^)]*\)\s*\{[^}]*return\s+redirect\(\s*['"]([^'"]+)['"]\s*\)[^}]*\}\s*\)"#
        ).unwrap();

        for caps in re_redirect_closure.captures_iter(content) {
            let method = caps[1].to_uppercase();
            let path = &caps[2];
            let target_url = caps[3].to_string();

            let full_path = format!("{}/{}", prefix, path.trim_matches('/'));
            let clean_path = if full_path.starts_with('/') { full_path } else { format!("/{}", full_path) };

            routes.push(self.create_route(&method, &clean_path, RouteHandler::ClosureRedirect(target_url)));
        }

        routes
    }

    fn create_route(&self, method: &str, path: &str, handler: RouteHandler) -> DynamicRoute {
        let clean_path = if path.starts_with('/') { path.to_string() } else { format!("/{}", path) };

        // Parse param names e.g. {id}
        let re_param = Regex::new(r"\{([a-zA-Z0-9_]+)\}").unwrap();
        let mut param_names = Vec::new();
        for caps in re_param.captures_iter(&clean_path) {
            param_names.push(caps[1].to_string());
        }

        // Convert {param} to ([^/]+) regex capture group
        let regex_pattern = format!("^{}$", re_param.replace_all(&clean_path, "([^/]+)"));
        let regex = Regex::new(&regex_pattern).unwrap();

        DynamicRoute {
            method: method.to_string(),
            path_pattern: clean_path,
            regex,
            param_names,
            handler,
        }
    }
}
