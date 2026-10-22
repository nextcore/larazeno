use serde_json::Value;
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

/// Blade AST Node
#[derive(Debug, Clone, PartialEq)]
pub enum BladeNode {
    /// Plain HTML / CSS / JS text
    Text(String),
    /// Comment: {{-- ... --}}
    Comment(String),
    /// Variable echo escaped: {{ $var }} or {{ $var.key }} or {{ $var['key'] }}
    EchoEscaped(String),
    /// Raw unescaped echo: {!! $var !!}
    EchoRaw(String),
    /// Helper call echo: {{ config('app.name') }} or {{ app()->version() }}
    EchoHelper {
        expr: String,
    },
    /// Directive: @directive(args)
    Directive {
        name: String,
        arg: Option<String>,
    },
    /// If conditional: @if (cond) children [@else children] @endif
    If {
        condition: String,
        then_branch: Vec<BladeNode>,
        else_branch: Option<Vec<BladeNode>>,
    },
    /// Foreach loop: @foreach ($items as $item) children @endforeach
    Foreach {
        collection: String,
        item_var: String,
        body: Vec<BladeNode>,
    },
}

/// Token types for Blade Lexer
#[derive(Debug, Clone, PartialEq)]
enum BladeToken {
    Text(String),
    Comment(String),
    EchoEscaped(String),
    EchoRaw(String),
    DirectiveOpen { name: String, arg: Option<String> },
}

pub struct BladeParser;

impl BladeParser {
    /// Daftar directive standar Laravel
    const KNOWN_DIRECTIVES: &'static [&'static str] = &[
        "if",
        "else",
        "elseif",
        "endif",
        "foreach",
        "endforeach",
        "for",
        "endfor",
        "while",
        "endwhile",
        "extends",
        "section",
        "endsection",
        "yield",
        "include",
        "csrf",
        "method",
        "auth",
        "endauth",
        "guest",
        "endguest",
        "vite",
        "fonts",
        "viteReactRefresh",
        "inertia",
        "push",
        "endpush",
        "stack",
        "props",
    ];

    /// Tokenizer: memecah input string menjadi stream Token Blade berbasis char index
    fn tokenize(input: &str) -> Vec<BladeToken> {
        let mut tokens = Vec::new();
        let chars: Vec<char> = input.chars().collect();
        let len = chars.len();
        let mut cursor = 0;

        let mut current_text = String::new();

        while cursor < len {
            // 1. Blade comment: {{-- ... --}}
            if cursor + 4 <= len
                && chars[cursor] == '{'
                && chars[cursor + 1] == '{'
                && chars[cursor + 2] == '-'
                && chars[cursor + 3] == '-'
            {
                if !current_text.is_empty() {
                    tokens.push(BladeToken::Text(std::mem::take(&mut current_text)));
                }
                cursor += 4;
                let start_comment = cursor;
                let mut found_end = false;
                while cursor + 4 <= len {
                    if chars[cursor] == '-'
                        && chars[cursor + 1] == '-'
                        && chars[cursor + 2] == '}'
                        && chars[cursor + 3] == '}'
                    {
                        let comment_str: String = chars[start_comment..cursor].iter().collect();
                        tokens.push(BladeToken::Comment(comment_str));
                        cursor += 4;
                        found_end = true;
                        break;
                    }
                    cursor += 1;
                }
                if !found_end {
                    let comment_str: String = chars[start_comment..].iter().collect();
                    tokens.push(BladeToken::Comment(comment_str));
                    cursor = len;
                }
                continue;
            }

            // 2. Raw Echo: {!! ... !!}
            if cursor + 3 <= len
                && chars[cursor] == '{'
                && chars[cursor + 1] == '!'
                && chars[cursor + 2] == '!'
            {
                if !current_text.is_empty() {
                    tokens.push(BladeToken::Text(std::mem::take(&mut current_text)));
                }
                cursor += 3;
                let start_echo = cursor;
                let mut found_end = false;
                while cursor + 3 <= len {
                    if chars[cursor] == '!' && chars[cursor + 1] == '!' && chars[cursor + 2] == '}' {
                        let expr: String = chars[start_echo..cursor].iter().collect();
                        tokens.push(BladeToken::EchoRaw(expr.trim().to_string()));
                        cursor += 3;
                        found_end = true;
                        break;
                    }
                    cursor += 1;
                }
                if !found_end {
                    cursor = len;
                }
                continue;
            }

            // 3. Escaped Echo: {{ ... }}
            if cursor + 2 <= len && chars[cursor] == '{' && chars[cursor + 1] == '{' {
                if !current_text.is_empty() {
                    tokens.push(BladeToken::Text(std::mem::take(&mut current_text)));
                }
                cursor += 2;
                let start_echo = cursor;
                let mut found_end = false;
                while cursor + 2 <= len {
                    if chars[cursor] == '}' && chars[cursor + 1] == '}' {
                        let expr: String = chars[start_echo..cursor].iter().collect();
                        tokens.push(BladeToken::EchoEscaped(expr.trim().to_string()));
                        cursor += 2;
                        found_end = true;
                        break;
                    }
                    cursor += 1;
                }
                if !found_end {
                    cursor = len;
                }
                continue;
            }

            // 4. Directive: @name or @name(...)
            // Identifikasi nama directive setelah simbol '@'
            if chars[cursor] == '@' && cursor + 1 < len && chars[cursor + 1].is_ascii_alphabetic() {
                let start_name = cursor + 1;
                let mut end_name = start_name;
                while end_name < len && (chars[end_name].is_ascii_alphanumeric() || chars[end_name] == '_') {
                    end_name += 1;
                }
                let potential_name: String = chars[start_name..end_name].iter().collect();

                // Pastikan directive adalah known Blade directive (bukan CSS @media, @layer, @keyframes, @supports, dsb.)
                if Self::KNOWN_DIRECTIVES.contains(&potential_name.as_str()) {
                    if !current_text.is_empty() {
                        tokens.push(BladeToken::Text(std::mem::take(&mut current_text)));
                    }
                    cursor = end_name;

                    // Parse optional argument: @directive(...)
                    let mut arg = None;
                    let mut peek = cursor;
                    while peek < len && chars[peek].is_ascii_whitespace() {
                        peek += 1;
                    }
                    if peek < len && chars[peek] == '(' {
                        cursor = peek + 1;
                        let start_arg = cursor;
                        let mut paren_depth = 1;
                        while cursor < len && paren_depth > 0 {
                            if chars[cursor] == '(' {
                                paren_depth += 1;
                            } else if chars[cursor] == ')' {
                                paren_depth -= 1;
                                if paren_depth == 0 {
                                    break;
                                }
                            }
                            cursor += 1;
                        }
                        let arg_str: String = chars[start_arg..cursor].iter().collect();
                        arg = Some(arg_str.trim().to_string());
                        if cursor < len && chars[cursor] == ')' {
                            cursor += 1;
                        }
                    }

                    tokens.push(BladeToken::DirectiveOpen {
                        name: potential_name,
                        arg,
                    });
                    continue;
                }
            }

            // Karakter teks normal (HTML, CSS, JS, whitespace)
            current_text.push(chars[cursor]);
            cursor += 1;
        }

        if !current_text.is_empty() {
            tokens.push(BladeToken::Text(current_text));
        }

        tokens
    }

    /// AST Parser: Membangun Blade AST Tree dengan Recursive Descent
    pub fn parse(input: &str) -> Vec<BladeNode> {
        let tokens = Self::tokenize(input);
        let mut idx = 0;
        Self::parse_nodes(&tokens, &mut idx, None)
    }

    fn parse_nodes(
        tokens: &[BladeToken],
        idx: &mut usize,
        until_directive: Option<&[&str]>,
    ) -> Vec<BladeNode> {
        let mut nodes = Vec::new();

        while *idx < tokens.len() {
            match &tokens[*idx] {
                BladeToken::Text(t) => {
                    nodes.push(BladeNode::Text(t.clone()));
                    *idx += 1;
                }
                BladeToken::Comment(c) => {
                    nodes.push(BladeNode::Comment(c.clone()));
                    *idx += 1;
                }
                BladeToken::EchoRaw(r) => {
                    nodes.push(BladeNode::EchoRaw(r.clone()));
                    *idx += 1;
                }
                BladeToken::EchoEscaped(expr) => {
                    if expr.starts_with('$') {
                        nodes.push(BladeNode::EchoEscaped(expr.clone()));
                    } else {
                        nodes.push(BladeNode::EchoHelper { expr: expr.clone() });
                    }
                    *idx += 1;
                }
                BladeToken::DirectiveOpen { name, arg } => {
                    // Check if closing directive matching `until_directive`
                    if let Some(stop_words) = until_directive {
                        if stop_words.contains(&name.as_str()) {
                            return nodes;
                        }
                    }

                    // Foreach block: @foreach (...) ... @endforeach
                    if name == "foreach" {
                        let arg_str = arg.clone().unwrap_or_default();
                        // Parse "$products as $product"
                        let mut coll = String::new();
                        let mut item = String::new();
                        if let Some((left, right)) = arg_str.split_once(" as ") {
                            coll = left.trim().trim_start_matches('$').to_string();
                            item = right.trim().trim_start_matches('$').to_string();
                        }

                        *idx += 1; // consume @foreach
                        let body = Self::parse_nodes(tokens, idx, Some(&["endforeach"]));
                        if *idx < tokens.len() {
                            if let BladeToken::DirectiveOpen { name, .. } = &tokens[*idx] {
                                if name == "endforeach" {
                                    *idx += 1; // consume @endforeach
                                }
                            }
                        }

                        nodes.push(BladeNode::Foreach {
                            collection: coll,
                            item_var: item,
                            body,
                        });
                        continue;
                    }

                    // If block: @if (...) ... [@else ...] @endif
                    if name == "if" {
                        let condition = arg.clone().unwrap_or_default();
                        *idx += 1; // consume @if
                        let then_branch = Self::parse_nodes(tokens, idx, Some(&["else", "endif"]));

                        let mut else_branch = None;
                        if *idx < tokens.len() {
                            if let BladeToken::DirectiveOpen { name, .. } = &tokens[*idx] {
                                if name == "else" {
                                    *idx += 1; // consume @else
                                    else_branch = Some(Self::parse_nodes(tokens, idx, Some(&["endif"])));
                                }
                            }
                        }

                        if *idx < tokens.len() {
                            if let BladeToken::DirectiveOpen { name, .. } = &tokens[*idx] {
                                if name == "endif" {
                                    *idx += 1; // consume @endif
                                }
                            }
                        }

                        nodes.push(BladeNode::If {
                            condition,
                            then_branch,
                            else_branch,
                        });
                        continue;
                    }

                    // Standalone directives (@csrf, @method, @fonts, @vite, dsb.)
                    nodes.push(BladeNode::Directive {
                        name: name.clone(),
                        arg: arg.clone(),
                    });
                    *idx += 1;
                }
            }
        }

        nodes
    }
}

/// Evaluator: Berjalan di atas Blade AST Nodes untuk menghasilkan rendered HTML
pub struct BladeEvaluator;

impl BladeEvaluator {
    pub fn evaluate(nodes: &[BladeNode], data: &Value, scope: &HashMap<String, Value>) -> String {
        let mut out = String::new();

        for node in nodes {
            match node {
                BladeNode::Text(t) => {
                    out.push_str(t);
                }
                BladeNode::Comment(_) => {
                    // Blade comments are stripped from compiled HTML output
                }
                BladeNode::EchoRaw(expr) => {
                    let val = Self::resolve_expr(expr, data, scope);
                    out.push_str(&val);
                }
                BladeNode::EchoEscaped(expr) => {
                    let val = Self::resolve_expr(expr, data, scope);
                    out.push_str(&Self::escape_html(&val));
                }
                BladeNode::EchoHelper { expr } => {
                    let val = Self::resolve_helper_expr(expr);
                    out.push_str(&val);
                }
                BladeNode::Directive { name, arg } => {
                    // Standard Laravel directives
                    match name.as_str() {
                        "csrf" => {
                            // CSRF Token input field jika berada di dalam form HTML
                            out.push_str(r#"<input type="hidden" name="_token" value="larazeno-ast-token">"#);
                        }
                        "method" => {
                            // Method spoofing: @method('DELETE') atau @method('PUT')
                            let method_val = arg.as_deref().unwrap_or("").trim().trim_matches('\'').trim_matches('"');
                            if !method_val.is_empty() {
                                out.push_str(&format!(
                                    r#"<input type="hidden" name="_method" value="{}">"#,
                                    method_val
                                ));
                            }
                        }
                        "fonts" => {
                            // Fonts directive placeholder
                        }
                        _ => {
                            // Directive lain yang tidak merender DOM
                        }
                    }
                }
                BladeNode::If {
                    condition,
                    then_branch,
                    else_branch,
                } => {
                    if Self::eval_condition(condition) {
                        out.push_str(&Self::evaluate(then_branch, data, scope));
                    } else if let Some(else_b) = else_branch {
                        out.push_str(&Self::evaluate(else_b, data, scope));
                    }
                }
                BladeNode::Foreach {
                    collection,
                    item_var,
                    body,
                } => {
                    // Cari koleksi data dari scope lokal atau root data
                    let coll_val = scope.get(collection).or_else(|| data.get(collection));
                    if let Some(items) = coll_val.and_then(|v| v.as_array()) {
                        for item in items {
                            let mut local_scope = scope.clone();
                            local_scope.insert(item_var.clone(), item.clone());
                            out.push_str(&Self::evaluate(body, data, &local_scope));
                        }
                    }
                }
            }
        }

        out
    }

    /// Evaluasi kondisi @if (misal: file_exists, Route::has)
    fn eval_condition(condition: &str) -> bool {
        let cond = condition.trim();

        // 1. file_exists(public_path('build/manifest.json')) -> false saat vite dev belum build
        if cond.starts_with("file_exists") {
            return false;
        }

        // 2. Route::has('login') / Route::has('register')
        if cond.contains("Route::has") {
            return false;
        }

        false
    }

    /// Evaluasi ekspresi variabel: $product.id, $product['id'], $var
    fn resolve_expr(expr: &str, data: &Value, scope: &HashMap<String, Value>) -> String {
        let clean = expr.trim();
        let var_expr = clean.trim_start_matches('$');

        // Bentuk: product.id atau product['id'] atau product["id"]
        if let Some((root, rest)) = var_expr.split_once('.') {
            let key = rest.trim();
            if let Some(val) = scope.get(root).or_else(|| data.get(root)) {
                if let Some(field) = val.get(key) {
                    return Self::value_to_string(field);
                }
            }
            return String::new();
        }

        if let Some((root, rest)) = var_expr.split_once('[') {
            let key = rest.trim_end_matches(']').trim().trim_matches('\'').trim_matches('"');
            if let Some(val) = scope.get(root).or_else(|| data.get(root)) {
                if let Some(field) = val.get(key) {
                    return Self::value_to_string(field);
                }
            }
            return String::new();
        }

        // Direct variable name
        if let Some(val) = scope.get(var_expr).or_else(|| data.get(var_expr)) {
            return Self::value_to_string(val);
        }

        String::new()
    }

    /// Evaluasi Helper functions seperti config('app.name'), app()->getLocale(), app()->version()
    fn resolve_helper_expr(expr: &str) -> String {
        let clean = expr.trim();

        if clean.contains("config('app.name'") || clean.contains("config(\"app.name\"") {
            return "Laravel".to_string();
        }

        if clean.contains("app()->getLocale()") || clean.contains("getLocale()") {
            return "en".to_string();
        }

        if clean.contains("app()->version()") {
            return "13.0.0 (LaraZeno-rs Pure Rust AST)".to_string();
        }

        String::new()
    }

    fn value_to_string(val: &Value) -> String {
        match val {
            Value::String(s) => s.clone(),
            Value::Number(n) => n.to_string(),
            Value::Bool(b) => b.to_string(),
            _ => String::new(),
        }
    }

    fn escape_html(s: &str) -> String {
        let mut escaped = String::with_capacity(s.len());
        for c in s.chars() {
            match c {
                '&' => escaped.push_str("&amp;"),
                '<' => escaped.push_str("&lt;"),
                '>' => escaped.push_str("&gt;"),
                '"' => escaped.push_str("&quot;"),
                '\'' => escaped.push_str("&#039;"),
                _ => escaped.push(c),
            }
        }
        escaped
    }
}

pub struct BladeEngine {
    views_path: PathBuf,
}

impl BladeEngine {
    pub fn new(project_path: &Path) -> Self {
        Self {
            views_path: project_path.join("resources").join("views"),
        }
    }

    /// Render Blade template menggunakan Full AST Lexer, Parser, dan Evaluator
    pub fn render(&self, view_name: &str, data: &Value) -> Result<String, String> {
        let rel_path = format!("{}.blade.php", view_name.replace('.', "/"));
        let full_path = self.views_path.join(&rel_path);

        let content = fs::read_to_string(&full_path)
            .map_err(|e| format!("View {} not found at {:?}: {}", view_name, full_path, e))?;

        // 1. Build AST Nodes from raw template
        let ast = BladeParser::parse(&content);

        // 2. Evaluate AST with Scope & Data
        let scope = HashMap::new();
        let rendered_html = BladeEvaluator::evaluate(&ast, data, &scope);

        Ok(rendered_html)
    }
}
