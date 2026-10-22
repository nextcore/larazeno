package inertia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"larazeno-core/pkg/blade"
	"larazeno-core/pkg/engine"
	"larazeno-core/pkg/vendors"
)

type InertiaProvider struct{}

func (i *InertiaProvider) PackageName() string {
	return "inertiajs/inertia-laravel"
}

func init() {
	vendors.Register(&InertiaProvider{})
}

func (i *InertiaProvider) RegisterSlots(eng *engine.Engine, appCtx *vendors.AppContext) error {

	// SLOT: inertia.render (Menggantikan Inertia::render('Component', $props))
	eng.Register("inertia.render", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		wVal, ok := scope.Get("http.response_writer")
		if !ok {
			return fmt.Errorf("http.response_writer not found")
		}
		w := wVal.(http.ResponseWriter)
		rVal, _ := scope.Get("http.request")

		component := fmt.Sprintf("%v", node.Value)
		props := make(map[string]interface{})

		// 1. Ambil seluruh variabel yang ada di scope controller/closure secara universal
		if scope != nil {
			for k, v := range scope.All() {
				// Lewati internal engine state variables (dimulai dengan underscore atau http)
				if !strings.HasPrefix(k, "_") && !strings.HasPrefix(k, "http.") && k != "auth.user" {
					props[k] = v
				}
			}
		}

		// 2. Evaluasi props eksplisit yang dideklarasikan di Inertia::render('Component', ['key' => $val])
		for _, c := range node.Children {
			if c.Name == "prop" {
				propKey := fmt.Sprintf("%v", c.Value)
				var resolvedVal interface{} = map[string]interface{}{}

				if len(c.Children) > 0 {
					rawVal := fmt.Sprintf("%v", c.Children[0].Value)
					if strings.HasPrefix(rawVal, "$") {
						varName := strings.TrimPrefix(rawVal, "$")
						if val, exists := scope.Get(varName); exists {
							resolvedVal = val
						}
					} else if rawVal != "" {
						resolvedVal = rawVal
					}
				}

				if val, exists := props[propKey]; exists && val != nil {
					// jika sudah ada dari scope, pertahankan
				} else {
					props[propKey] = resolvedVal
				}
			}
		}

		currentURL := "/"
		// Cek apakah request datang dari frontend Inertia (header X-Inertia)
		isInertiaRequest := false
		if rVal != nil {
			r := rVal.(*http.Request)
			currentURL = r.URL.RequestURI()
			if r.Header.Get("X-Inertia") == "true" {
				isInertiaRequest = true
			}
		}

		if isInertiaRequest {
			// Kirim JSON Payload Inertia
			responsePayload := map[string]interface{}{
				"component": component,
				"props":     props,
				"url":       currentURL,
				"version":   "1.0",
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Inertia", "true")
			w.WriteHeader(http.StatusOK)
			return json.NewEncoder(w).Encode(responsePayload)
		}

		// Jika request pertama kali dari browser biasa (Full Page Visit), sajikan HTML shell app.blade.php
		bladeEng := blade.NewBladeEngine(appCtx.ProjectRoot)
		htmlContent, err := bladeEng.Render("app", map[string]interface{}{})
		if err != nil {
			htmlContent = `<!DOCTYPE html><html><head><title>LaraZeno Inertia</title></head><body>@inertia</body></html>`
		}

		// Siapkan data-page JSON
		pageData, _ := json.Marshal(map[string]interface{}{
			"component": component,
			"props":     props,
			"url":       currentURL,
			"version":   "1.0",
		})

		pageDataAttr := strings.ReplaceAll(string(pageData), "'", "&#039;")
		// Inertia v2.x menggunakan <script data-page="app" type="application/json">...</script>
		// Sementara Inertia v1.x menggunakan <div id="app" data-page="..."></div>
		// Kita sediakan keduanya agar kompatibel 100% dengan semua versi Inertia:
		appMarkup := fmt.Sprintf(`<script data-page="app" type="application/json">%s</script><div id="app" data-page='%s'></div>`, string(pageData), pageDataAttr)

		// Replace @inertia & helpers
		htmlContent = strings.ReplaceAll(htmlContent, "@inertia", appMarkup)
		
		hotFile := filepath.Join(appCtx.ProjectRoot, "public", "hot")
		if _, err := os.Stat(hotFile); err == nil {
			htmlContent = strings.ReplaceAll(htmlContent, "@viteReactRefresh", `<script type="module">
  import RefreshRuntime from 'http://127.0.0.1:5173/@react-refresh'
  RefreshRuntime.injectIntoGlobalHook(window)
  window.$RefreshReg$ = () => {}
  window.$RefreshSig$ = () => (type) => type
  window.__vite_plugin_react_preamble_installed__ = true
</script>`)
		} else {
			// Pada production build (tanpa Vite dev server aktif), @viteReactRefresh harus kosong agar tidak memblokir render
			htmlContent = strings.ReplaceAll(htmlContent, "@viteReactRefresh", "")
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(htmlContent))
		return err
	})

	return nil
}
