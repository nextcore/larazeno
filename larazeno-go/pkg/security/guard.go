package security

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// DangerousExtensions adalah daftar ekstensi skrip yang dilarang keras dieksekusi langsung
var DangerousExtensions = map[string]bool{
	".php":   true,
	".php3":  true,
	".php4":  true,
	".php5":  true,
	".php7":  true,
	".php8":  true,
	".phtml": true,
	".phar":  true,
	".sh":    true,
	".bash":  true,
	".pl":    true,
	".py":    true,
	".rb":    true,
	".cgi":   true,
	".env":   true,
	".git":   true,
}

// AllowedStaticExts adalah ekstensi aman yang boleh diakses dari folder public
var AllowedStaticExts = map[string]string{
	".css":   "text/css",
	".js":    "application/javascript",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".webp":  "image/webp",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".txt":   "text/plain",
	".json":  "application/json",
}

// GuardDirectScriptExecution memblokir request langsung ke skrip ilegal/berbahaya
func GuardDirectScriptExecution(w http.ResponseWriter, r *http.Request) bool {
	cleanPath := strings.ToLower(r.URL.Path)

	// 1. Cek upaya akses ke file sensitif (.env, .git, .htaccess)
	if strings.Contains(cleanPath, ".env") || strings.Contains(cleanPath, ".git") || strings.Contains(cleanPath, ".htaccess") {
		log.Printf("🚨 [SECURITY ALERT] Unauthorized sensitive file access blocked: %s (IP: %s)", r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, `{"error":"Forbidden","message":"Direct access to sensitive configuration files is strictly blocked by LaraZeno Security Guard."}`)
		return true
	}

	// 2. Cek apakah request mencoba memanggil file skrip ilegal sembarangan (e.g. /coba.php, /shell.php, /uploads/backdoor.php)
	ext := strings.ToLower(filepath.Ext(cleanPath))
	if DangerousExtensions[ext] {
		log.Printf("🛡️ [SECURITY GUARD] Blocked illegal script execution attempt: %s (IP: %s)", r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, `{"error":"Forbidden","message":"Direct script execution (%s) is prohibited by LaraZeno. Only authorized Laravel routes can be executed."}`, ext)
		return true
	}

	return false
}
