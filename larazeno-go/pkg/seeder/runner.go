package seeder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"larazeno-core/pkg/orm"
)

// Runner menangani eksekusi seeder database Laravel
type Runner struct {
	projectPath string
}

// NewRunner membuat instance seeder runner baru
func NewRunner(projectPath string) *Runner {
	return &Runner{
		projectPath: projectPath,
	}
}

// Run menjalankan seluruh seeder di folder database/seeders/
func (r *Runner) Run() error {
	seedersDir := filepath.Join(r.projectPath, "database", "seeders")
	if _, err := os.Stat(seedersDir); os.IsNotExist(err) {
		return fmt.Errorf("folder seeders tidak ditemukan di %s", seedersDir)
	}

	fmt.Println("🌱 Menjalankan Seeder Database LaraZeno...")
	fmt.Println("----------------------------------------------------------------")

	// 1. Eksekusi file-file seeder spesifik terlebih dahulu jika ada
	var seederFiles []string
	entries, err := os.ReadDir(seedersDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "Seeder.php") {
			// Jalankan DatabaseSeeder paling akhir agar dependency-nya terpenuhi
			if entry.Name() != "DatabaseSeeder.php" {
				seederFiles = append(seederFiles, entry.Name())
			}
		}
	}
	// Tambahkan DatabaseSeeder jika ada
	if _, err := os.Stat(filepath.Join(seedersDir, "DatabaseSeeder.php")); err == nil {
		seederFiles = append(seederFiles, "DatabaseSeeder.php")
	}

	totalInserted := 0
	for _, filename := range seederFiles {
		fullPath := filepath.Join(seedersDir, filename)
		count, err := r.executeSeederFile(fullPath)
		if err != nil {
			return fmt.Errorf("gagal mengeksekusi seeder %s: %w", filename, err)
		}
		if count > 0 {
			fmt.Printf("   ✔  %s: %d record ditambahkan\n", filename, count)
			totalInserted += count
		} else {
			fmt.Printf("   ✔  %s: (diproses)\n", filename)
		}
	}

	fmt.Println("----------------------------------------------------------------")
	fmt.Printf("✅ Seeding selesai! Total %d record berhasil di-insert.\n\n", totalInserted)
	return nil
}

// executeSeederFile membaca file PHP seeder dan mengeksekusi Model::create([...])
func (r *Runner) executeSeederFile(filePath string) (int, error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	content := string(contentBytes)
	insertedCount := 0

	// Regex untuk mencocokkan pattern: ModelName::create([...])
	createRegex := regexp.MustCompile(`([A-Z][a-zA-Z0-9_]*)::create\s*\(\s*\[([\s\S]*?)\]\s*\)`)
	matches := createRegex.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		modelName := m[1]
		body := m[2]

		data := parseArrayKeyValues(body)
		if len(data) > 0 {
			_, err := orm.Create(modelName, data)
			if err != nil {
				return insertedCount, fmt.Errorf("gagal insert %s: %w", modelName, err)
			}
			insertedCount++
		}
	}

	return insertedCount, nil
}

// parseArrayKeyValues mem-parsing pasangan 'key' => 'val' atau 'key' => 123
func parseArrayKeyValues(body string) map[string]interface{} {
	data := make(map[string]interface{})
	kvRegex := regexp.MustCompile(`['"]([a-zA-Z0-9_]+)['"]\s*=>\s*(?:['"]([^'"]*)['"]|([0-9]+(?:\.[0-9]+)?))`)
	matches := kvRegex.FindAllStringSubmatch(body, -1)

	for _, m := range matches {
		key := m[1]
		if m[2] != "" {
			// String value
			data[key] = m[2]
		} else if m[3] != "" {
			// Numeric value
			if num, err := strconv.Atoi(m[3]); err == nil {
				data[key] = num
			} else if floatVal, err := strconv.ParseFloat(m[3], 64); err == nil {
				data[key] = floatVal
			} else {
				data[key] = m[3]
			}
		}
	}

	return data
}
