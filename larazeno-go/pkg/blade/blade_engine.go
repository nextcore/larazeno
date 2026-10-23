package blade

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BladeEngine bertugas merender file .blade.php menjadi HTML menggunakan data scope Go
type BladeEngine struct {
	ProjectRoot string
	ViewsPath   string
}

func NewBladeEngine(projectRoot string) *BladeEngine {
	return &BladeEngine{
		ProjectRoot: projectRoot,
		ViewsPath:   filepath.Join(projectRoot, "resources", "views"),
	}
}

// Render memproses file blade (termasuk inheritance @extends, @section/@yield, @foreach, @if, dan {{ $var }})
func (b *BladeEngine) Render(viewName string, data map[string]interface{}) (string, error) {
	relPath := strings.ReplaceAll(viewName, ".", "/") + ".blade.php"
	fullPath := filepath.Join(b.ViewsPath, relPath)

	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("blade view not found: %s (%w)", fullPath, err)
	}

	raw := string(contentBytes)

	// 1. Handle Layout Inheritance (@extends dan @section)
	reExtends := regexp.MustCompile(`@extends\(['"]([^'"]+)['"]\)`)
	if match := reExtends.FindStringSubmatch(raw); len(match) > 1 {
		layoutName := match[1]
		layoutRelPath := strings.ReplaceAll(layoutName, ".", "/") + ".blade.php"
		layoutFullPath := filepath.Join(b.ViewsPath, layoutRelPath)

		layoutBytes, err := os.ReadFile(layoutFullPath)
		if err == nil {
			layoutContent := string(layoutBytes)

			reSection := regexp.MustCompile(`(?s)@section\(['"]([^'"]+)['"]\)(.*?)@endsection`)
			sections := reSection.FindAllStringSubmatch(raw, -1)
			for _, sec := range sections {
				secName := sec[1]
				secBody := sec[2]
				layoutContent = strings.ReplaceAll(layoutContent, fmt.Sprintf("@yield('%s')", secName), secBody)
				layoutContent = strings.ReplaceAll(layoutContent, fmt.Sprintf(`@yield("%s")`, secName), secBody)
			}
			raw = layoutContent
		}
	}

	// 1.5 Handle Partial Views (@include('partials.name'))
	reInclude := regexp.MustCompile(`@include\(['"]([^'"]+)['"]\)`)
	raw = reInclude.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reInclude.FindStringSubmatch(m)
		if len(sub) > 1 {
			includeName := sub[1]
			includeRelPath := strings.ReplaceAll(includeName, ".", "/") + ".blade.php"
			includeFullPath := filepath.Join(b.ViewsPath, includeRelPath)
			if incBytes, err := os.ReadFile(includeFullPath); err == nil {
				return string(incBytes)
			}
		}
		return ""
	})

	// 2. Handle Loops (@foreach($items as $item) ... @endforeach)
	reForeach := regexp.MustCompile(`(?s)@foreach\(\$([a-zA-Z0-9_]+)\s+as\s+\$([a-zA-Z0-9_]+)\)(.*?)@endforeach`)
	raw = reForeach.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reForeach.FindStringSubmatch(m)
		listVarName := sub[1]
		itemVarName := sub[2]
		loopBody := sub[3]

		var output strings.Builder
		if listVal, ok := data[listVarName]; ok {
			switch items := listVal.(type) {
			case []map[string]interface{}:
				for _, itm := range items {
					itemRendered := loopBody
					// Replace {{ $item['field'] }} atau {{ $item.field }}
					// Replace {{ $item['field'] }} atau {{ $item.field }}
					for k, v := range itm {
						valStr := fmt.Sprintf("%v", v)
						if v == nil {
							valStr = ""
						}
						itemRendered = strings.ReplaceAll(itemRendered, fmt.Sprintf("{{ $%s['%s'] }}", itemVarName, k), html.EscapeString(valStr))
						itemRendered = strings.ReplaceAll(itemRendered, fmt.Sprintf("{{ $%s.%s }}", itemVarName, k), html.EscapeString(valStr))
						itemRendered = strings.ReplaceAll(itemRendered, fmt.Sprintf("{{$%s['%s']}}", itemVarName, k), html.EscapeString(valStr))
						itemRendered = strings.ReplaceAll(itemRendered, fmt.Sprintf("{{$%s.%s}}", itemVarName, k), html.EscapeString(valStr))
					}

					// Process @if(!empty($item['field'])) or @if(!empty($item.field))
					reIfNotEmpty := regexp.MustCompile(fmt.Sprintf(`(?s)@if\(!empty\(\$%s(?:\['([^']+)'\]|\.([a-zA-Z0-9_]+))\)\)(.*?)@else(.*?)@endif`, itemVarName))
					itemRendered = reIfNotEmpty.ReplaceAllStringFunc(itemRendered, func(m string) string {
						sub := reIfNotEmpty.FindStringSubmatch(m)
						field := sub[1]
						if field == "" {
							field = sub[2]
						}
						val, exists := itm[field]
						if exists && val != nil && fmt.Sprintf("%v", val) != "" {
							return sub[3] // true branch
						}
						return sub[4] // false branch
					})

					// Process @if(!empty(...)) without @else
					reIfNotEmptySingle := regexp.MustCompile(fmt.Sprintf(`(?s)@if\(!empty\(\$%s(?:\['([^']+)'\]|\.([a-zA-Z0-9_]+))\)\)(.*?)@endif`, itemVarName))
					itemRendered = reIfNotEmptySingle.ReplaceAllStringFunc(itemRendered, func(m string) string {
						sub := reIfNotEmptySingle.FindStringSubmatch(m)
						field := sub[1]
						if field == "" {
							field = sub[2]
						}
						val, exists := itm[field]
						if exists && val != nil && fmt.Sprintf("%v", val) != "" {
							return sub[3]
						}
						return ""
					})

					output.WriteString(itemRendered)
				}
			}
		}
		return output.String()
	})

	// 3. Handle Variable Echo: {{ $var }} (Escaped) & {!! $var !!} (Raw HTML)
	reEchoEscaped := regexp.MustCompile(`\{\{\s*\$([a-zA-Z0-9_]+)\s*\}\}`)
	raw = reEchoEscaped.ReplaceAllStringFunc(raw, func(m string) string {
		varName := reEchoEscaped.FindStringSubmatch(m)[1]
		if val, ok := data[varName]; ok {
			return html.EscapeString(fmt.Sprintf("%v", val))
		}
		return m
	})

	reEchoRaw := regexp.MustCompile(`\{!!\s*\$([a-zA-Z0-9_]+)\s*!!\}`)
	raw = reEchoRaw.ReplaceAllStringFunc(raw, func(m string) string {
		varName := reEchoRaw.FindStringSubmatch(m)[1]
		if val, ok := data[varName]; ok {
			return fmt.Sprintf("%v", val)
		}
		return m
	})

	// 4. Handle @vite directive: @vite([...]) or @vite('...')
	reVite := regexp.MustCompile(`@vite\((.*?)\)`)
	raw = reVite.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reVite.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		argsStr := strings.TrimSpace(sub[1])
		// Extract paths inside quotes
		reArg := regexp.MustCompile(`['"]([^'"]+)['"]`)
		matches := reArg.FindAllStringSubmatch(argsStr, -1)
		var entrypoints []string
		for _, match := range matches {
			if len(match) > 1 {
				entrypoints = append(entrypoints, match[1])
			}
		}

		return b.renderViteTags(entrypoints)
	})

	// 5. Handle Generic Laravel Helper Functions in Blade: {{ config('app.name', 'Laravel') }}, {{ app()->getLocale() }}
	reConfigHelper := regexp.MustCompile(`\{\{\s*config\(['"]([^'"]+)['"](?:\s*,\s*['"]([^'"]*)['"])?\)\s*\}\}`)
	raw = reConfigHelper.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reConfigHelper.FindStringSubmatch(m)
		key := sub[1]
		defaultVal := ""
		if len(sub) > 2 {
			defaultVal = sub[2]
		}
		if key == "app.name" {
			if defaultVal != "" {
				return defaultVal
			}
			return "Laravel"
		}
		return defaultVal
	})

	reLocaleHelper := regexp.MustCompile(`\{\{\s*str_replace\(['"][^'"]*['"]\s*,\s*['"][^'"]*['"]\s*,\s*app\(\)->getLocale\(\)\)\s*\}\}`)
	raw = reLocaleHelper.ReplaceAllString(raw, "en")

	reAppVersion := regexp.MustCompile(`\{\{\s*app\(\)->version\(\)\s*\}\}`)
	raw = reAppVersion.ReplaceAllString(raw, "13.0.0 (LaraZeno Engine)")

	reAppLocale := regexp.MustCompile(`\{\{\s*app\(\)->getLocale\(\)\s*\}\}`)
	raw = reAppLocale.ReplaceAllString(raw, "en")

	// Handle Blade comments: {{-- ... --}}
	reBladeComments := regexp.MustCompile(`(?s)\{\{--.*?--\}\}`)
	raw = reBladeComments.ReplaceAllString(raw, "")

	// 6. Handle @fonts directive
	raw = strings.ReplaceAll(raw, "@fonts", "")

	// 7. Handle conditional blocks: @if (Route::has('login')) -> false by default if not set
	reIfRoute := regexp.MustCompile(`(?s)@if\s*\(\s*Route::has\([^)]+\)\s*\)(.*?)@endif`)
	raw = reIfRoute.ReplaceAllString(raw, "")

	// Handle @if condition for @vite fallback styling
	reIfVite := regexp.MustCompile(`(?s)@if\s*\(\s*file_exists\(public_path\([^)]+\)\).*?\)(.*?)@else(.*?)@endif`)
	raw = reIfVite.ReplaceAllStringFunc(raw, func(m string) string {
		sub := reIfVite.FindStringSubmatch(m)
		if len(sub) > 2 {
			return sub[2] // Gunakan fallback inline style (@else)
		}
		return m
	})

	// 8. Clean up any remaining unknown directives
	reCleanup := regexp.MustCompile(`@(csrf|method\([^)]+\)|endif|endauth|else|auth)`)
	raw = reCleanup.ReplaceAllString(raw, "")

	return raw, nil
}

// renderViteTags merender script tag untuk dev mode atau production build
func (b *BladeEngine) renderViteTags(entrypoints []string) string {
	// 1. Cek apakah ada file public/hot (Vite dev server aktif)
	hotFile := filepath.Join(b.ProjectRoot, "public", "hot")
	devServerURL := "http://127.0.0.1:5173"

	if hotBytes, err := os.ReadFile(hotFile); err == nil {
		url := strings.TrimSpace(string(hotBytes))
		if url != "" {
			devServerURL = url
		}
	}

	var sb strings.Builder
	// In dev mode (default if no manifest exists or hot file is present)
	manifestPath := filepath.Join(b.ProjectRoot, "public", "build", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		// Vite Dev Server tags
		sb.WriteString(fmt.Sprintf(`<script type="module" src="%s/@vite/client"></script>`+"\n", devServerURL))
		for _, ep := range entrypoints {
			if strings.HasSuffix(ep, ".css") {
				sb.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="%s/%s" />`+"\n", devServerURL, ep))
			} else {
				sb.WriteString(fmt.Sprintf(`<script type="module" src="%s/%s"></script>`+"\n", devServerURL, ep))
			}
		}
		return sb.String()
	}

	// Production manifest mode: Parse public/build/manifest.json
	manifestBytes, err := os.ReadFile(manifestPath)
	if err == nil {
		type ManifestItem struct {
			File string   `json:"file"`
			CSS  []string `json:"css"`
		}
		var manifest map[string]ManifestItem
		if err := json.Unmarshal(manifestBytes, &manifest); err == nil {
			for _, ep := range entrypoints {
				if item, ok := manifest[ep]; ok {
					// Add entry CSS imports first if any
					for _, cssFile := range item.CSS {
						sb.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="/build/%s" />`+"\n", cssFile))
					}
					// Add entry file
					if strings.HasSuffix(item.File, ".css") {
						sb.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="/build/%s" />`+"\n", item.File))
					} else {
						sb.WriteString(fmt.Sprintf(`<script type="module" src="/build/%s"></script>`+"\n", item.File))
					}
				}
			}
			return sb.String()
		}
	}

	for _, ep := range entrypoints {
		if strings.HasSuffix(ep, ".css") {
			sb.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="/build/%s" />`+"\n", ep))
		} else {
			sb.WriteString(fmt.Sprintf(`<script type="module" src="/build/%s"></script>`+"\n", ep))
		}
	}
	return sb.String()
}
