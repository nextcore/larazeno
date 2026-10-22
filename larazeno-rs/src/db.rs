use rusqlite::{Connection, Result};
use serde_json::{Map, Value};
use std::sync::{Arc, Mutex};

#[derive(Clone)]
pub struct DbPool {
    conn: Arc<Mutex<Connection>>,
}

impl DbPool {
    pub fn new(path: &str) -> Result<Self> {
        let conn = Connection::open(path)?;
        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
        })
    }

    pub fn execute(&self, sql: &str) -> Result<usize> {
        let conn = self.conn.lock().unwrap();
        conn.execute(sql, [])
    }

    pub fn all(&self, table: &str) -> Result<Vec<Value>> {
        let conn = self.conn.lock().unwrap();
        let query = format!("SELECT * FROM {}", table);
        let mut stmt = conn.prepare(&query)?;
        
        let col_names: Vec<String> = stmt.column_names().into_iter().map(String::from).collect();
        
        let rows = stmt.query_map([], |row| {
            let mut map = Map::new();
            for (idx, col) in col_names.iter().enumerate() {
                let val: Result<i64, _> = row.get(idx);
                if let Ok(v) = val {
                    map.insert(col.clone(), Value::Number(serde_json::Number::from(v)));
                    continue;
                }
                let val_str: Result<String, _> = row.get(idx);
                if let Ok(v) = val_str {
                    map.insert(col.clone(), Value::String(v));
                    continue;
                }
                map.insert(col.clone(), Value::Null);
            }
            Ok(Value::Object(map))
        })?;

        let mut results = Vec::new();
        for r in rows {
            if let Ok(v) = r {
                results.push(v);
            }
        }
        Ok(results)
    }

    pub fn insert(&self, table: &str, data: &Value) -> Result<i64> {
        let conn = self.conn.lock().unwrap();
        if let Some(obj) = data.as_object() {
            let cols: Vec<String> = obj.keys().cloned().collect();
            let placeholders: Vec<&str> = cols.iter().map(|_| "?").collect();
            let sql = format!(
                "INSERT INTO {} ({}) VALUES ({})",
                table,
                cols.join(", "),
                placeholders.join(", ")
            );

            let mut params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();
            for col in &cols {
                match &obj[col] {
                    Value::Number(n) => {
                        if let Some(i) = n.as_i64() {
                            params.push(Box::new(i));
                        } else if let Some(f) = n.as_f64() {
                            params.push(Box::new(f));
                        }
                    }
                    Value::String(s) => params.push(Box::new(s.clone())),
                    _ => params.push(Box::new(Option::<String>::None)),
                }
            }

            let params_slice: Vec<&dyn rusqlite::ToSql> = params.iter().map(|p| p.as_ref()).collect();
            conn.execute(&sql, params_slice.as_slice())?;
            return Ok(conn.last_insert_rowid());
        }
        Ok(0)
    }

    pub fn update(&self, table: &str, id: i64, data: &Value) -> Result<usize> {
        let conn = self.conn.lock().unwrap();
        if let Some(obj) = data.as_object() {
            let mut set_clauses = Vec::new();
            let mut params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

            for (k, v) in obj {
                if k == "id" || k == "_method" {
                    continue;
                }
                set_clauses.push(format!("{} = ?", k));
                match v {
                    Value::Number(n) => {
                        if let Some(i) = n.as_i64() {
                            params.push(Box::new(i));
                        } else if let Some(f) = n.as_f64() {
                            params.push(Box::new(f));
                        }
                    }
                    Value::String(s) => {
                        if let Ok(num) = s.parse::<i64>() {
                            if k == "price" || k == "stock" {
                                params.push(Box::new(num));
                            } else {
                                params.push(Box::new(s.clone()));
                            }
                        } else {
                            params.push(Box::new(s.clone()));
                        }
                    }
                    _ => params.push(Box::new(Option::<String>::None)),
                }
            }

            params.push(Box::new(id));
            let sql = format!("UPDATE {} SET {} WHERE id = ?", table, set_clauses.join(", "));
            let params_slice: Vec<&dyn rusqlite::ToSql> = params.iter().map(|p| p.as_ref()).collect();
            return conn.execute(&sql, params_slice.as_slice());
        }
        Ok(0)
    }

    pub fn delete(&self, table: &str, id: i64) -> Result<usize> {
        let conn = self.conn.lock().unwrap();
        let sql = format!("DELETE FROM {} WHERE id = ?", table);
        conn.execute(&sql, [id])
    }
}
