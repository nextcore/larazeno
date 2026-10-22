package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"larazeno-core/pkg/db"
)

// ColumnDefinition merepresentasikan definisi kolom pada Blueprint
type ColumnDefinition struct {
	Name       string
	Type       string
	IsPrimary  bool
	IsNullable bool
	IsUnique   bool
	DefaultVal string
}

// TableDefinition merepresentasikan tabel yang di-create oleh Schema::create
type TableDefinition struct {
	Name    string
	Columns []ColumnDefinition
}

// Runner menangani eksekusi migrasi database Laravel
type Runner struct {
	projectPath string
	driver      string
}

// NewRunner membuat instance migration runner baru
func NewRunner(projectPath string) *Runner {
	driver := "sqlite3"
	if db.DBConn != nil {
		driver = db.DBConn.Driver
	}
	return &Runner{
		projectPath: projectPath,
		driver:      driver,
	}
}

// EnsureMigrationTable memastikan tabel tracking migrasi ada
func (r *Runner) EnsureMigrationTable() error {
	var sql string
	if r.driver == "mysql" || r.driver == "mariadb" {
		sql = `
		CREATE TABLE IF NOT EXISTS migrations (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL
		) ENGINE=InnoDB;`
	} else {
		sql = `
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			migration TEXT NOT NULL,
			batch INTEGER NOT NULL
		);`
	}

	_, err := db.DBConn.DB.Exec(sql)
	return err
}

// GetAppliedMigrations mengambil daftar nama file migrasi yang sudah pernah dijalankan
func (r *Runner) GetAppliedMigrations() (map[string]bool, int, error) {
	applied := make(map[string]bool)
	maxBatch := 0

	rows, err := db.DBConn.DB.Query("SELECT migration, batch FROM migrations ORDER BY id ASC")
	if err != nil {
		return applied, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var batch int
		if err := rows.Scan(&name, &batch); err == nil {
			applied[name] = true
			if batch > maxBatch {
				maxBatch = batch
			}
		}
	}

	return applied, maxBatch, nil
}

// Run mengeksekusi semua migrasi yang belum dijalankan
func (r *Runner) Run() error {
	if err := r.EnsureMigrationTable(); err != nil {
		return fmt.Errorf("gagal inisialisasi tabel migrations: %w", err)
	}

	applied, maxBatch, err := r.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("gagal membaca tabel migrations: %w", err)
	}

	migrationsDir := filepath.Join(r.projectPath, "database", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("folder database/migrations tidak ditemukan di %s", migrationsDir)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".php") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	currentBatch := maxBatch + 1
	executedCount := 0

	fmt.Println("🚀 Menjalankan Migrasi Database LaraZeno...")
	fmt.Printf("   Driver: %s\n", r.driver)
	fmt.Println("----------------------------------------------------------------")

	for _, file := range files {
		if applied[file] {
			continue
		}

		filePath := filepath.Join(migrationsDir, file)
		contentBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("gagal membaca file migrasi %s: %w", file, err)
		}

		tables := parseMigrationFile(string(contentBytes))
		if len(tables) == 0 {
			// Jika tidak ada Schema::create, catat saja migrasi selesai
			fmt.Printf("   ℹ  %s (tidak ada Schema::create terdeteksi)\n", file)
		}

		for _, tbl := range tables {
			ddl := generateCreateTableSQL(r.driver, tbl)
			if _, err := db.DBConn.DB.Exec(ddl); err != nil {
				return fmt.Errorf("gagal migrasi %s pada tabel '%s': %w\nSQL: %s", file, tbl.Name, err, ddl)
			}
			fmt.Printf("   ✔  Tabel '%s' dibuat/diverifikasi\n", tbl.Name)
		}

		// Catat ke tabel migrations
		_, err = db.DBConn.DB.Exec("INSERT INTO migrations (migration, batch) VALUES (?, ?)", file, currentBatch)
		if err != nil {
			return fmt.Errorf("gagal mencatat status migrasi %s: %w", file, err)
		}

		fmt.Printf("   DONE: %s\n", file)
		executedCount++
	}

	fmt.Println("----------------------------------------------------------------")
	if executedCount == 0 {
		fmt.Println("✨ Tidak ada migrasi baru untuk dijalankan (Database sudah up to date).")
	} else {
		fmt.Printf("✅ Sukses mengeksekusi %d migrasi (Batch %d).\n", executedCount, currentBatch)
	}

	return nil
}

// Status menampilkan status migrasi yang sudah atau belum dijalankan
func (r *Runner) Status() error {
	if err := r.EnsureMigrationTable(); err != nil {
		return err
	}

	applied, _, err := r.GetAppliedMigrations()
	if err != nil {
		return err
	}

	migrationsDir := filepath.Join(r.projectPath, "database", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("folder database/migrations tidak ditemukan di %s", migrationsDir)
	}

	fmt.Println("📋 Status Migrasi Database:")
	fmt.Printf("%-50s %-10s\n", "Nama File Migrasi", "Status")
	fmt.Println(strings.Repeat("-", 62))

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".php") {
			status := "Pending"
			if applied[entry.Name()] {
				status = "Ran (✔)"
			}
			fmt.Printf("%-50s %-10s\n", entry.Name(), status)
		}
	}
	return nil
}

// parseMigrationFile mengekstrak Schema::create(...) dari file migrasi PHP
func parseMigrationFile(content string) []TableDefinition {
	var tables []TableDefinition

	// Regex mencocokkan Schema::create('nama_tabel', function (...) { ... })
	schemaCreateRegex := regexp.MustCompile(`(?s)Schema::create\s*\(\s*['"]([^'"]+)['"]\s*,\s*function\s*\([^\)]*\)\s*\{([^}]+)\}`)
	matches := schemaCreateRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		tableName := match[1]
		body := match[2]

		table := TableDefinition{
			Name: tableName,
		}

		// Parse setiap baris $table->...
		colLines := strings.Split(body, ";")
		for _, rawLine := range colLines {
			line := strings.TrimSpace(rawLine)
			if line == "" || !strings.HasPrefix(line, "$table->") {
				continue
			}

			col := parseColumnChain(line)
			if col != nil {
				if col.Type == "morphs" {
					// morphs('tokenable') menghasilkan 2 kolom: tokenable_type dan tokenable_id
					tablesMorph := []ColumnDefinition{
						{Name: col.Name + "_type", Type: "string"},
						{Name: col.Name + "_id", Type: "bigInteger"},
					}
					table.Columns = append(table.Columns, tablesMorph...)
				} else if col.Type == "timestamps" {
					table.Columns = append(table.Columns,
						ColumnDefinition{Name: "created_at", Type: "timestamp", IsNullable: true},
						ColumnDefinition{Name: "updated_at", Type: "timestamp", IsNullable: true},
					)
				} else if col.Type == "rememberToken" {
					table.Columns = append(table.Columns,
						ColumnDefinition{Name: "remember_token", Type: "string", IsNullable: true},
					)
				} else {
					table.Columns = append(table.Columns, *col)
				}
			}
		}

		tables = append(tables, table)
	}

	return tables
}

// parseColumnChain mem-parse chaining method seperti $table->string('email')->unique()->nullable()
func parseColumnChain(line string) *ColumnDefinition {
	chain := strings.TrimPrefix(line, "$table->")

	// Panggil method pertama: id(), string('name'), dll
	methodRegex := regexp.MustCompile(`^([a-zA-Z0-9_]+)\s*\((.*?)\)`)
	matches := methodRegex.FindStringSubmatch(chain)
	if len(matches) < 2 {
		return nil
	}

	methodType := matches[1]
	argsRaw := matches[2]

	col := &ColumnDefinition{
		Type: methodType,
	}

	// Ekstrak nama kolom dari argumen pertama jika ada
	argRegex := regexp.MustCompile(`['"]([^'"]+)['"]`)
	argMatches := argRegex.FindAllStringSubmatch(argsRaw, -1)
	if len(argMatches) > 0 {
		col.Name = argMatches[0][1]
	} else if methodType == "id" {
		col.Name = "id"
		col.IsPrimary = true
	} else if methodType == "timestamps" || methodType == "rememberToken" {
		col.Name = methodType
	} else {
		// Abaikan pemanggilan method seperti index() atau foreign() tanpa nama
		return nil
	}

	// Cek modifiers
	if strings.Contains(chain, "->primary()") {
		col.IsPrimary = true
	}
	if strings.Contains(chain, "->nullable()") {
		col.IsNullable = true
	}
	if strings.Contains(chain, "->unique()") {
		col.IsUnique = true
	}

	return col
}

// generateCreateTableSQL menyusun sintaks DDL CREATE TABLE IF NOT EXISTS
func generateCreateTableSQL(driver string, table TableDefinition) string {
	var colDefs []string
	isMysql := (driver == "mysql" || driver == "mariadb")

	for _, col := range table.Columns {
		def := ""
		sqlType := ""

		switch col.Type {
		case "id":
			if isMysql {
				sqlType = "BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY"
			} else {
				sqlType = "INTEGER PRIMARY KEY AUTOINCREMENT"
			}
		case "string", "uuid":
			if isMysql {
				sqlType = "VARCHAR(255)"
			} else {
				sqlType = "TEXT"
			}
		case "text", "mediumText", "longText":
			if isMysql {
				if col.Type == "longText" {
					sqlType = "LONGTEXT"
				} else if col.Type == "mediumText" {
					sqlType = "MEDIUMTEXT"
				} else {
					sqlType = "TEXT"
				}
			} else {
				sqlType = "TEXT"
			}
		case "integer", "unsignedInteger", "unsignedSmallInteger":
			if isMysql {
				sqlType = "INT"
				if strings.Contains(col.Type, "unsigned") {
					sqlType += " UNSIGNED"
				}
			} else {
				sqlType = "INTEGER"
			}
		case "bigInteger", "foreignId":
			if isMysql {
				sqlType = "BIGINT UNSIGNED"
			} else {
				sqlType = "INTEGER"
			}
		case "boolean":
			if isMysql {
				sqlType = "TINYINT(1)"
			} else {
				sqlType = "INTEGER"
			}
		case "timestamp", "dateTime":
			if isMysql {
				sqlType = "TIMESTAMP NULL"
			} else {
				sqlType = "DATETIME"
			}
		default:
			if isMysql {
				sqlType = "VARCHAR(255)"
			} else {
				sqlType = "TEXT"
			}
		}

		def = fmt.Sprintf("`%s` %s", col.Name, sqlType)
		if !isMysql {
			def = fmt.Sprintf("\"%s\" %s", col.Name, sqlType)
		}

		if !strings.Contains(sqlType, "PRIMARY KEY") {
			if col.IsPrimary {
				def += " PRIMARY KEY"
			}
			if col.IsUnique {
				def += " UNIQUE"
			}
			if !col.IsNullable && !strings.Contains(sqlType, "NULL") {
				def += " NOT NULL"
			}
		}

		colDefs = append(colDefs, def)
	}

	engineSuffix := ""
	if isMysql {
		engineSuffix = " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)%s;",
		table.Name,
		strings.Join(colDefs, ",\n  "),
		engineSuffix,
	)
}
