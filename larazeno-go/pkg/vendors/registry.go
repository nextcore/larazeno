package vendors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"larazeno-core/pkg/engine"
	"larazeno-core/pkg/router"
)

// AppContext menyimpan state runtime aplikasi yang dibagi antar vendors
type AppContext struct {
	DynamicRouter *router.DynamicRouter
	Controllers   map[string]*engine.Node // "UserController" -> class AST
	ProjectRoot   string
	Packages      map[string]string // "laravel/framework": "^13.17", "laravel/sanctum": "^4.0"
	Env           map[string]string
}

func NewAppContext(projectRoot string) *AppContext {
	return &AppContext{
		DynamicRouter: router.NewDynamicRouter(),
		Controllers:   make(map[string]*engine.Node),
		ProjectRoot:   projectRoot,
		Packages:      make(map[string]string),
		Env:           make(map[string]string),
	}
}

// VendorProvider adalah interface yang harus diimplementasikan oleh setiap Go Virtual Vendor
type VendorProvider interface {
	PackageName() string // misal "laravel/framework" atau "laravel/sanctum"
	RegisterSlots(eng *engine.Engine, appCtx *AppContext) error
}

// Registry mengelola daftar semua VendorProvider yang tersedia di LaraZeno
type Registry struct {
	providers map[string]VendorProvider
}

var GlobalRegistry = &Registry{
	providers: make(map[string]VendorProvider),
}

// Register mendaftarkan vendor provider ke registry global
func Register(provider VendorProvider) {
	GlobalRegistry.providers[provider.PackageName()] = provider
}

// LoadFromComposer membaca require dari composer.json dan me-load slot vendor yang cocok
func LoadFromComposer(eng *engine.Engine, appCtx *AppContext) error {
	composerPath := filepath.Join(appCtx.ProjectRoot, "composer.json")
	data, err := os.ReadFile(composerPath)
	if err != nil {
		return fmt.Errorf("gagal membaca composer.json: %w", err)
	}

	var comp struct {
		Require map[string]string `json:"require"`
	}
	if err := json.Unmarshal(data, &comp); err != nil {
		return fmt.Errorf("gagal unmarshal composer.json: %w", err)
	}

	appCtx.Packages = comp.Require
	fmt.Println("\n📦 [Vendor Loader] Scanning composer.json packages:")

	// Selalu load laravel/framework secara default jika ada deklarasi php/laravel
	for pkgName, version := range comp.Require {
		if provider, exists := GlobalRegistry.providers[pkgName]; exists {
			fmt.Printf("   ✔ Activating Go Virtual Vendor: %s (%s)\n", pkgName, version)
			if err := provider.RegisterSlots(eng, appCtx); err != nil {
				return err
			}
		} else {
			// Paket yang belum ada slot Go nya (atau murni php util)
			fmt.Printf("   ℹ Skipping unmapped package: %s (%s)\n", pkgName, version)
		}
	}

	// Always ensure laravel/framework provider is registered if not explicitly triggered
	if _, ok := comp.Require["laravel/framework"]; !ok {
		if fw, ok := GlobalRegistry.providers["laravel/framework"]; ok {
			_ = fw.RegisterSlots(eng, appCtx)
		}
	}

	return nil
}

// Dummy context use
var _ = context.Background
