package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"larazeno-core/pkg/config"
	"larazeno-core/pkg/db"
	"larazeno-core/pkg/engine"
	"larazeno-core/pkg/parser"
	"larazeno-core/pkg/security"
	"larazeno-core/pkg/slots/sanctum"
	"larazeno-core/pkg/vendors"

	// Register virtual vendor providers
	_ "larazeno-core/pkg/vendors/inertia"
	_ "larazeno-core/pkg/vendors/laravel/framework"
	_ "larazeno-core/pkg/vendors/laravel/sanctum"
)

func main() {
	// 1. Argument CLI: Menentukan project path & port
	projectPathFlag := flag.String("path", "", "Path ke folder project Laravel")
	portFlag := flag.String("port", "9001", "Port server LaraZeno")
	flag.Parse()

	laravelProjectPath := *projectPathFlag
	if laravelProjectPath == "" {
		cwd, err := os.Getwd()
		if err == nil {
			laravelProjectPath = cwd
		}
	}
	if !fileExists(filepath.Join(laravelProjectPath, "artisan")) {
		log.Printf("⚠️  Peringatan: file 'artisan' tidak ditemukan di %s. Pastikan flag -path mengarah ke direktori proyek Laravel yang valid.", laravelProjectPath)
	}

	fmt.Println("================================================================")
	fmt.Println("  LaraZeno: Generic Laravel 13 Execution Engine (Pure Go AST)   ")
	fmt.Println("  Zero Composer | Zero Vendor Folder | Universal REST API       ")
	fmt.Println("================================================================")
	fmt.Printf("📁 Project Root: %s\n", laravelProjectPath)

	// 2. Baca Konfigurasi .env Proyek Secara Dinamis
	env, err := config.LoadEnv(laravelProjectPath)
	if err != nil {
		log.Printf("⚠️  .env tidak ditemukan atau gagal dibaca (%v), menggunakan default SQLite", err)
	}

	var dbDriver, dbDSN string
	if env != nil && (env.DBConnection == "mysql" || env.DBConnection == "mariadb") {
		dbDriver = "mysql"
		port := env.DBPort
		if port == "" {
			port = "3306"
		}
		dbDSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			env.DBUsername, env.DBPassword, env.DBHost, port, env.DBDatabase)
	} else {
		dbDriver = "sqlite3"
		if env != nil && env.DBDatabase != "" {
			dbDSN = env.DBDatabase
		} else {
			dbDSN = filepath.Join(laravelProjectPath, "database", "database.sqlite")
		}
	}

	_, err = db.Connect(dbDriver, dbDSN)
	if err != nil {
		log.Fatalf("❌ Gagal konek Database (%s: %s): %v", dbDriver, dbDSN, err)
	}

	// 3. Inisialisasi Engine Core & Auto-load Vendor Slots dari composer.json
	eng := engine.NewEngine()
	appCtx := vendors.NewAppContext(laravelProjectPath)
	if env != nil && env.All != nil {
		appCtx.Env = env.All
	}
	if err := vendors.LoadFromComposer(eng, appCtx); err != nil {
		log.Fatalf("❌ Gagal load virtual vendors: %v", err)
	}

	// 4. Auto-Discovery & Parse Semua Controller di app/Http/Controllers
	controllersDir := filepath.Join(laravelProjectPath, "app", "Http", "Controllers")
	controllerCount := 0
	_ = filepath.WalkDir(controllersDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), "Controller.php") {
			cAST, err := parser.ParsePHPFile(path)
			if err == nil {
				for _, child := range cAST.Children {
					if child.Name == "php.class" {
						_ = eng.Execute(context.Background(), child, engine.NewScope(nil))
					}
				}
				fmt.Printf("✔ Loaded Controller: %s\n", d.Name())
				controllerCount++
			}
		}
		return nil
	})
	fmt.Printf("📦 Total Controllers Loaded: %d\n", controllerCount)

	// 5. Auto-Discovery & Parse semua file rute di routes/ (api.php, web.php, auth.php, dll)
	routesDir := filepath.Join(laravelProjectPath, "routes")
	_ = filepath.WalkDir(routesDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".php") && d.Name() != "console.php" {
			rAST, err := parser.ParsePHPFile(path)
			if err == nil {
				_ = eng.Execute(context.Background(), rAST, engine.NewScope(nil))
				fmt.Printf("✔ Loaded Routes File: %s\n", d.Name())
			}
		}
		return nil
	})
	fmt.Printf("✔ Total Routes Registered: %d\n", len(appCtx.DynamicRouter.Routes))

	// 6. Generic HTTP Router & Dispatcher (Tidak ada rute yang di-hardcode)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// [SECURITY LAYER] Cegah eksekusi skrip ilegal langsung (.php, .sh, webshell, file sensitif .env)
		if security.GuardDirectScriptExecution(w, r) {
			return
		}

		method := r.Method
		path := r.URL.Path

		// Match rute secara dinamis via regex pattern matching
		matchedRoute, params := appCtx.DynamicRouter.Match(method, path)
		if matchedRoute == nil {
			// Cek apakah static file ada di folder public/ (misal /build/assets/..., /favicon.ico)
			cleanPath := filepath.Clean(strings.TrimPrefix(path, "/"))
			publicFile := filepath.Join(appCtx.ProjectRoot, "public", cleanPath)
			if stat, err := os.Stat(publicFile); err == nil && !stat.IsDir() {
				http.ServeFile(w, r, publicFile)
				return
			}

			http.NotFound(w, r)
			return
		}

		// 1. Evaluasi Middleware (auth:sanctum)
		var authUser map[string]interface{}
		if matchedRoute.Middleware == "auth:sanctum" {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated (Sanctum Bearer Token Required)."})
				return
			}
			bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
			_, user, err := sanctum.AuthenticateToken(bearerToken)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated (Invalid Sanctum Token)."})
				return
			}
			authUser = user
		}

		// 2. Cari Controller & Method di AST
		parts := strings.Split(matchedRoute.Action, "@")
		if len(parts) != 2 {
			http.Error(w, "Invalid action handler", 500)
			return
		}
		ctrlName := parts[0]
		methodName := parts[1]

		if ctrlName == "Closure" {
			// Eksekusi Closure secara dinamis dari AST yang di-parse
			closureScope := engine.NewScope(nil)
			closureScope.Set("http.request", r)
			closureScope.Set("http.response_writer", w)
			for k, v := range params {
				closureScope.Set(k, v)
			}
			ctx := context.Background()

			// Cari action node di HandlerNode route
			var closureAction *engine.Node
			if matchedRoute.HandlerNode != nil {
				for _, c := range matchedRoute.HandlerNode.Children {
					if c.Name == "action" {
						closureAction = c
						break
					}
				}
			}

			if closureAction != nil && len(closureAction.Children) > 0 {
				for _, stmt := range closureAction.Children {
					_ = eng.Execute(ctx, stmt, closureScope)
				}
			}
			return
		}

		ctrlNode, exists := appCtx.Controllers[ctrlName]
		if !exists {
			http.Error(w, fmt.Sprintf("Controller %s not found", ctrlName), 404)
			return
		}

		var targetMethod *engine.Node
		for _, c := range ctrlNode.Children {
			if c.Name == "php.method" && fmt.Sprintf("%v", c.Value) == methodName {
				targetMethod = c
				break
			}
		}
		if targetMethod == nil {
			http.Error(w, fmt.Sprintf("Method %s not found on %s", methodName, ctrlName), 404)
			return
		}

		// 3. Buat Scope dan pass dynamic route parameters
		scope := engine.NewScope(nil)
		scope.Set("http.request", r)
		scope.Set("http.response_writer", w)
		if authUser != nil {
			scope.Set("auth.user", authUser)
		}
		for k, v := range params {
			scope.Set(k, v)
		}

		// 4. Eksekusi AST method controller
		ctx := context.Background()
		for _, stmt := range targetMethod.Children {
			if stmt.Name != "arg" {
				if err := eng.Execute(ctx, stmt, scope); err != nil {
					http.Error(w, fmt.Sprintf("Execution Error: %v", err), 500)
					return
				}
			}
		}

		// 5. Fallback jika controller action adalah redirect namun belum terkirim response
		if _, sent := scope.Get("_response_sent"); !sent {
			if r.Method == "POST" && r.Header.Get("X-Inertia") == "true" {
				target := r.Header.Get("Referer")
				if target == "" {
					target = "/"
				}
				http.Redirect(w, r, target, http.StatusSeeOther)
				return
			}
		}
	})

	listenPort := ":" + *portFlag
	fmt.Printf("\n🌐 LaraZeno Server listening on http://127.0.0.1%s\n", listenPort)
	fmt.Println("   Ready to serve dynamic Laravel REST API endpoints.")
	fmt.Println("================================================================")

	if err := http.ListenAndServe(listenPort, nil); err != nil {
		log.Fatal(err)
	}
}


func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
