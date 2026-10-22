package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	Driver string
	DB     *sql.DB
}

var DBConn *Database

// Connect menginisialisasi koneksi SQLite atau MariaDB/MySQL
func Connect(driver, dsn string) (*Database, error) {
	var sqlDriver string

	switch driver {
	case "sqlite", "sqlite3":
		sqlDriver = "sqlite3"
		if dsn == "" {
			dsn = "./database/database.sqlite"
		}
	case "mysql", "mariadb":
		sqlDriver = "mysql"
		if dsn == "" {
			dsn = "root:password@tcp(127.0.0.1:3306)/laravel?parseTime=true"
		}
	default:
		return nil, fmt.Errorf("driver database tidak didukung: %s (gunakan 'sqlite' atau 'mariadb')", driver)
	}

	conn, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("koneksi database gagal: %w", err)
	}

	DBConn = &Database{
		Driver: driver,
		DB:     conn,
	}

	fmt.Printf(" [Database Pool] Terhubung ke %s (DSN: %s)\n", driver, dsn)
	return DBConn, nil
}

// EnsureTables memastikan tabel users dan personal_access_tokens siap untuk pengujian
func (d *Database) EnsureTables() error {
	var userTableSQL string
	var tokenTableSQL string

	if d.Driver == "sqlite" || d.Driver == "sqlite3" {
		userTableSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`

		tokenTableSQL = `
		CREATE TABLE IF NOT EXISTS personal_access_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tokenable_type TEXT NOT NULL,
			tokenable_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			abilities TEXT,
			last_used_at DATETIME,
			expires_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);`
	} else {
		// MariaDB / MySQL
		userTableSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		) ENGINE=InnoDB;`

		tokenTableSQL = `
		CREATE TABLE IF NOT EXISTS personal_access_tokens (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			tokenable_type VARCHAR(255) NOT NULL,
			tokenable_id BIGINT UNSIGNED NOT NULL,
			name VARCHAR(255) NOT NULL,
			token VARCHAR(64) UNIQUE NOT NULL,
			abilities TEXT NULL,
			last_used_at TIMESTAMP NULL,
			expires_at TIMESTAMP NULL,
			created_at TIMESTAMP NULL,
			updated_at TIMESTAMP NULL
		) ENGINE=InnoDB;`
	}

	if _, err := d.DB.Exec(userTableSQL); err != nil {
		return fmt.Errorf("gagal membuat tabel users: %w", err)
	}
	if _, err := d.DB.Exec(tokenTableSQL); err != nil {
		return fmt.Errorf("gagal membuat tabel personal_access_tokens: %w", err)
	}

	return nil
}

// GetDefaultDBPath mengembalikan path database default
func GetDefaultDBPath() string {
	defaultPath := "./database/database.sqlite"
	_ = os.MkdirAll("./database", 0755)
	return defaultPath
}
