use crate::db::DbPool;
use regex::Regex;
use std::collections::HashSet;
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone)]
pub struct ColumnDefinition {
    pub name: String,
    pub col_type: String,
    pub is_primary: bool,
    pub is_nullable: bool,
    pub is_unique: bool,
}

#[derive(Debug, Clone)]
pub struct TableDefinition {
    pub name: String,
    pub columns: Vec<ColumnDefinition>,
}

pub struct MigrationRunner {
    project_path: PathBuf,
    db: DbPool,
}

impl MigrationRunner {
    pub fn new(project_path: &Path, db: DbPool) -> Self {
        Self {
            project_path: project_path.to_path_buf(),
            db,
        }
    }

    pub fn ensure_migrations_table(&self) -> Result<(), rusqlite::Error> {
        let sql = "CREATE TABLE IF NOT EXISTS migrations (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            migration TEXT NOT NULL,
            batch INTEGER NOT NULL
        );";
        self.db.execute(sql)?;
        Ok(())
    }

    pub fn get_applied_migrations(&self) -> Result<(HashSet<String>, i64), rusqlite::Error> {
        let mut applied = HashSet::new();
        let mut max_batch = 0;

        if let Ok(rows) = self.db.all("migrations") {
            for row in rows {
                if let Some(obj) = row.as_object() {
                    if let Some(name) = obj.get("migration").and_then(|v| v.as_str()) {
                        applied.insert(name.to_string());
                    }
                    if let Some(batch) = obj.get("batch").and_then(|v| v.as_i64()) {
                        if batch > max_batch {
                            max_batch = batch;
                        }
                    }
                }
            }
        }

        Ok((applied, max_batch))
    }

    pub fn run(&self) -> Result<(), Box<dyn std::error::Error>> {
        self.ensure_migrations_table()?;
        let (applied, max_batch) = self.get_applied_migrations()?;

        let migrations_dir = self.project_path.join("database").join("migrations");
        if !migrations_dir.exists() {
            return Err(format!("Direktori migrasi tidak ditemukan di {:?}", migrations_dir).into());
        }

        let mut files = Vec::new();
        for entry in fs::read_dir(&migrations_dir)? {
            let entry = entry?;
            let path = entry.path();
            if path.is_file() {
                if let Some(fname) = path.file_name().and_then(|s| s.to_str()) {
                    if fname.ends_with(".php") {
                        files.push(fname.to_string());
                    }
                }
            }
        }
        files.sort();

        let current_batch = max_batch + 1;
        let mut executed_count = 0;

        println!("🚀 Menjalankan Migrasi Database LaraZeno-rs (Pure Rust)...");
        println!("   Driver: sqlite3");
        println!("----------------------------------------------------------------");

        for file in &files {
            if applied.contains(file) {
                continue;
            }

            let file_path = migrations_dir.join(file);
            let content = fs::read_to_string(&file_path)?;

            let tables = parse_migration_file(&content);
            for tbl in tables {
                let ddl = generate_create_table_sql(&tbl);
                self.db.execute(&ddl)?;
                println!("   ✔  Tabel '{}' dibuat/diverifikasi", tbl.name);
            }

            // Catat ke tabel migrations
            let insert_sql = format!(
                "INSERT INTO migrations (migration, batch) VALUES ('{}', {})",
                file, current_batch
            );
            self.db.execute(&insert_sql)?;

            println!("   DONE: {}", file);
            executed_count += 1;
        }

        println!("----------------------------------------------------------------");
        if executed_count == 0 {
            println!("✨ Tidak ada migrasi baru untuk dijalankan (Database sudah up to date).");
        } else {
            println!("✅ Sukses mengeksekusi {} migrasi (Batch {}).", executed_count, current_batch);
        }

        Ok(())
    }

    pub fn status(&self) -> Result<(), Box<dyn std::error::Error>> {
        self.ensure_migrations_table()?;
        let (applied, _) = self.get_applied_migrations()?;

        let migrations_dir = self.project_path.join("database").join("migrations");
        let mut files = Vec::new();
        for entry in fs::read_dir(&migrations_dir)? {
            let entry = entry?;
            let path = entry.path();
            if path.is_file() {
                if let Some(fname) = path.file_name().and_then(|s| s.to_str()) {
                    if fname.ends_with(".php") {
                        files.push(fname.to_string());
                    }
                }
            }
        }
        files.sort();

        println!("📋 Status Migrasi Database (LaraZeno-rs):");
        println!("{:<50} {:<10}", "Nama File Migrasi", "Status");
        println!("{}", "-".repeat(62));

        for file in files {
            let status = if applied.contains(&file) {
                "Ran (✔)"
            } else {
                "Pending"
            };
            println!("{:<50} {:<10}", file, status);
        }

        Ok(())
    }
}

fn parse_migration_file(content: &str) -> Vec<TableDefinition> {
    let mut tables = Vec::new();

    // Regex mencocokkan Schema::create('nama_tabel', function (...) { ... })
    let re_schema = Regex::new(r#"(?s)Schema::create\s*\(\s*['"]([^'"]+)['"]\s*,\s*function\s*\([^\)]*\)\s*\{([^}]+)\}"#).unwrap();

    for caps in re_schema.captures_iter(content) {
        let table_name = caps[1].to_string();
        let body = &caps[2];

        let mut table = TableDefinition {
            name: table_name,
            columns: Vec::new(),
        };

        for raw_line in body.split(';') {
            let line = raw_line.trim();
            if line.is_empty() || !line.starts_with("$table->") {
                continue;
            }

            if let Some(col) = parse_column_chain(line) {
                if col.col_type == "morphs" {
                    table.columns.push(ColumnDefinition {
                        name: format!("{}_type", col.name),
                        col_type: "string".to_string(),
                        is_primary: false,
                        is_nullable: false,
                        is_unique: false,
                    });
                    table.columns.push(ColumnDefinition {
                        name: format!("{}_id", col.name),
                        col_type: "bigInteger".to_string(),
                        is_primary: false,
                        is_nullable: false,
                        is_unique: false,
                    });
                } else if col.col_type == "timestamps" {
                    table.columns.push(ColumnDefinition {
                        name: "created_at".to_string(),
                        col_type: "timestamp".to_string(),
                        is_primary: false,
                        is_nullable: true,
                        is_unique: false,
                    });
                    table.columns.push(ColumnDefinition {
                        name: "updated_at".to_string(),
                        col_type: "timestamp".to_string(),
                        is_primary: false,
                        is_nullable: true,
                        is_unique: false,
                    });
                } else if col.col_type == "rememberToken" {
                    table.columns.push(ColumnDefinition {
                        name: "remember_token".to_string(),
                        col_type: "string".to_string(),
                        is_primary: false,
                        is_nullable: true,
                        is_unique: false,
                    });
                } else {
                    table.columns.push(col);
                }
            }
        }

        tables.push(table);
    }

    tables
}

fn parse_column_chain(line: &str) -> Option<ColumnDefinition> {
    let chain = line.trim_start_matches("$table->");

    let re_method = Regex::new(r#"^([a-zA-Z0-9_]+)\s*\((.*?)\)"#).unwrap();
    let caps = re_method.captures(chain)?;

    let method_type = caps[1].to_string();
    let args_raw = &caps[2];

    let re_arg = Regex::new(r#"['"]([^'"]+)['"]"#).unwrap();
    let col_name = if let Some(arg_caps) = re_arg.captures(args_raw) {
        arg_caps[1].to_string()
    } else if method_type == "id" {
        "id".to_string()
    } else if method_type == "timestamps" || method_type == "rememberToken" {
        method_type.clone()
    } else {
        return None;
    };

    let is_primary = method_type == "id" || chain.contains("->primary()");
    let is_nullable = chain.contains("->nullable()");
    let is_unique = chain.contains("->unique()");

    Some(ColumnDefinition {
        name: col_name,
        col_type: method_type,
        is_primary,
        is_nullable,
        is_unique,
    })
}

fn generate_create_table_sql(table: &TableDefinition) -> String {
    let mut col_defs = Vec::new();

    for col in &table.columns {
        let sql_type = match col.col_type.as_str() {
            "id" => "INTEGER PRIMARY KEY AUTOINCREMENT",
            "string" | "uuid" | "text" | "mediumText" | "longText" => "TEXT",
            "integer" | "unsignedInteger" | "unsignedSmallInteger" | "bigInteger" | "foreignId" | "boolean" => "INTEGER",
            "timestamp" | "dateTime" => "DATETIME",
            _ => "TEXT",
        };

        let mut def = format!("\"{}\" {}", col.name, sql_type);
        if !sql_type.contains("PRIMARY KEY") {
            if col.is_primary {
                def.push_str(" PRIMARY KEY");
            }
            if col.is_unique {
                def.push_str(" UNIQUE");
            }
            if !col.is_nullable && !sql_type.contains("NULL") {
                def.push_str(" NOT NULL");
            }
        }

        col_defs.push(def);
    }

    format!(
        "CREATE TABLE IF NOT EXISTS {} (\n  {}\n);",
        table.name,
        col_defs.join(",\n  ")
    )
}
