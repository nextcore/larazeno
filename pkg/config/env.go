package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type LaravelEnv struct {
	AppName      string
	AppEnv       string
	AppKey       string
	DBConnection string
	DBHost       string
	DBPort       string
	DBDatabase   string
	DBUsername   string
	DBPassword   string
	All          map[string]string
}

func LoadEnv(projectRoot string) (*LaravelEnv, error) {
	envPath := filepath.Join(projectRoot, ".env")
	file, err := os.Open(envPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			envMap[k] = v
		}
	}

	dbConn := envMap["DB_CONNECTION"]
	if dbConn == "" {
		dbConn = "sqlite"
	}

	dbDatabase := envMap["DB_DATABASE"]
	if dbConn == "sqlite" && dbDatabase == "" {
		dbDatabase = filepath.Join(projectRoot, "database", "database.sqlite")
	} else if dbConn == "sqlite" && !filepath.IsAbs(dbDatabase) {
		dbDatabase = filepath.Join(projectRoot, "database", dbDatabase)
	}

	return &LaravelEnv{
		AppName:      envMap["APP_NAME"],
		AppEnv:       envMap["APP_ENV"],
		AppKey:       envMap["APP_KEY"],
		DBConnection: dbConn,
		DBHost:       envMap["DB_HOST"],
		DBPort:       envMap["DB_PORT"],
		DBDatabase:   dbDatabase,
		DBUsername:   envMap["DB_USERNAME"],
		DBPassword:   envMap["DB_PASSWORD"],
		All:          envMap,
	}, nil
}
