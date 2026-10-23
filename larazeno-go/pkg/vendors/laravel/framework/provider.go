package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"larazeno-core/pkg/blade"
	"larazeno-core/pkg/engine"
	"larazeno-core/pkg/orm"
	"larazeno-core/pkg/vendors"
)

type FrameworkProvider struct{}

func (f *FrameworkProvider) PackageName() string {
	return "laravel/framework"
}

func init() {
	vendors.Register(&FrameworkProvider{})
}

func (f *FrameworkProvider) RegisterSlots(eng *engine.Engine, appCtx *vendors.AppContext) error {

	// 1. SLOT: laravel.route (Route::get / Route::post / Route::put / Route::delete)
	eng.Register("laravel.route", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		httpMethod := fmt.Sprintf("%v", node.Value)
		var path string
		var action string
		var middleware string

		var routeName string
		var groupPrefix string
		for _, c := range node.Children {
			if c.Name == "path" {
				path = fmt.Sprintf("%v", c.Value)
			}
			if c.Name == "action" {
				action = fmt.Sprintf("%v", c.Value)
			}
			if c.Name == "middleware" {
				middleware = fmt.Sprintf("%v", c.Value)
			}
			if c.Name == "route_name" {
				routeName = fmt.Sprintf("%v", c.Value)
			}
			if c.Name == "group_prefix" {
				groupPrefix = fmt.Sprintf("%v", c.Value)
			}
		}

		if groupPrefix != "" {
			trimmedGroup := strings.Trim(groupPrefix, "/")
			trimmedPath := strings.Trim(path, "/")
			if trimmedPath == "" {
				path = "/" + trimmedGroup
			} else {
				path = "/" + trimmedGroup + "/" + trimmedPath
			}
		}

		appCtx.DynamicRouter.AddRoute(httpMethod, path, action, middleware, node)
		if routeName != "" {
			cleanPath := path
			if !strings.HasPrefix(cleanPath, "/") {
				cleanPath = "/" + cleanPath
			}
			appCtx.DynamicRouter.NamedRoutes[routeName] = cleanPath
		}
		if middleware != "" {
			fmt.Printf("   [laravel/framework: Route] %-6s %s -> %s [Middleware: %s]\n", httpMethod, path, action, middleware)
		} else {
			fmt.Printf("   [laravel/framework: Route] %-6s %s -> %s\n", httpMethod, path, action)
		}
		return nil
	})

	// 2. SLOT: laravel.route_group (Route::middleware(...)->group(...))
	eng.Register("laravel.route_group", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		for _, child := range node.Children {
			if err := eng.Execute(ctx, child, scope); err != nil {
				return err
			}
		}
		return nil
	})

	// 3. SLOT: laravel.api_resource (Route::apiResource('resource', Controller::class))
	eng.Register("laravel.api_resource", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		var resName string
		var ctrlName string
		var middleware string

		for _, c := range node.Children {
			if c.Name == "resource_name" {
				resName = fmt.Sprintf("%v", c.Value)
			}
			if c.Name == "controller" {
				ctrlName = fmt.Sprintf("%v", c.Value)
			}
			if c.Name == "middleware" {
				middleware = fmt.Sprintf("%v", c.Value)
			}
		}

		basePath := "/" + strings.TrimPrefix(resName, "/")
		endpoints := []struct {
			method string
			path   string
			action string
		}{
			{"GET", basePath, ctrlName + "@index"},
			{"POST", basePath, ctrlName + "@store"},
			{"GET", basePath + "/{id}", ctrlName + "@show"},
			{"PUT", basePath + "/{id}", ctrlName + "@update"},
			{"DELETE", basePath + "/{id}", ctrlName + "@destroy"},
		}

		for _, ep := range endpoints {
			appCtx.DynamicRouter.AddRoute(ep.method, ep.path, ep.action, middleware, node)
			fmt.Printf("   [laravel/framework: Resource] %-6s %s -> %s [Middleware: %s]\n", ep.method, ep.path, ep.action, middleware)
		}
		return nil
	})

	// 4. SLOT: php.class (Registrasi AST Controller ke registry)
	eng.Register("php.class", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		className := fmt.Sprintf("%v", node.Value)
		appCtx.Controllers[className] = node
		return nil
	})

	// 4A. SLOT: php.try (Eksekusi blok try statement dengan penanganan catch)
	eng.Register("php.try", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		var catchBlock *engine.Node
		var tryChildren []*engine.Node

		for _, child := range node.Children {
			if child.Name == "php.catch" {
				catchBlock = child
			} else {
				tryChildren = append(tryChildren, child)
			}
		}

		var executionErr error
		for _, child := range tryChildren {
			if err := eng.Execute(ctx, child, scope); err != nil {
				executionErr = err
				break
			}
		}

		// Jika terjadi error di blok try dan ada catch block, eksekusi pernyataan di dalam catch block
		if executionErr != nil && catchBlock != nil {
			scope.Set("exception", executionErr.Error())
			for _, stmt := range catchBlock.Children {
				_ = eng.Execute(ctx, stmt, scope)
			}
			return nil
		}

		return executionErr
	})

	// 5. SLOT: laravel.model_call (Eloquent ORM Model::all, Model::find, Model::create)
	eng.Register("laravel.model_call", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		callTarget := fmt.Sprintf("%v", node.Value)
		parts := strings.Split(callTarget, "::")
		if len(parts) != 2 {
			return nil
		}
		modelName := parts[0]
		methodName := parts[1]

		// Model::all()
		if methodName == "all" {
			records, err := orm.All(modelName)
			if err != nil {
				scope.Set("_last_expr_result", []interface{}{})
				return nil
			}
			scope.Set("_last_expr_result", records)
			return nil
		}

		// Model::find($id)
		if methodName == "find" {
			var idVal interface{} = "1"
			if len(node.Children) > 0 {
				rawArg := fmt.Sprintf("%v", node.Children[0].Value)
				if strings.HasPrefix(rawArg, "$") {
					if val, found := scope.Get(strings.TrimPrefix(rawArg, "$")); found {
						idVal = val
					}
				} else {
					idVal = rawArg
				}
			}

			record, err := orm.Find(modelName, idVal)
			if err != nil {
				scope.Set("_last_expr_result", nil)
				return nil
			}
			record["_model_name"] = modelName
			scope.Set("_last_expr_result", record)
			scope.Set(strings.ToLower(modelName), record)
			return nil
		}

		// Model::create($data)
		if methodName == "create" {
			reqVal, _ := scope.Get("http.request")
			data := make(map[string]interface{})
			if reqVal != nil {
				req := reqVal.(*http.Request)
				if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
					_ = json.NewDecoder(req.Body).Decode(&data)
				} else if strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
					_ = req.ParseMultipartForm(32 << 20) // 32MB max
					if req.MultipartForm != nil {
						for k, v := range req.MultipartForm.Value {
							if k != "_token" && k != "_method" && len(v) > 0 {
								data[k] = v[0]
							}
						}
						for fileKey, fileHeaders := range req.MultipartForm.File {
							if len(fileHeaders) > 0 {
								fh := fileHeaders[0]
								uploadsDir := filepath.Join(appCtx.ProjectRoot, "public", "uploads")
								_ = os.MkdirAll(uploadsDir, 0755)
								ext := filepath.Ext(fh.Filename)
								saveFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
								destPath := filepath.Join(uploadsDir, saveFilename)
								
								src, err := fh.Open()
								if err == nil {
									dst, err := os.Create(destPath)
									if err == nil {
										_, _ = io.Copy(dst, src)
										_ = dst.Close()
										data[fileKey] = "/uploads/" + saveFilename
									}
									_ = src.Close()
								}
							}
						}
					}
				} else {
					_ = req.ParseForm()
					for k, v := range req.PostForm {
						if k != "_token" && k != "_method" && len(v) > 0 {
							data[k] = v[0]
						}
					}
				}
			}

			record, err := orm.Create(modelName, data)
			if err != nil {
				return err
			}
			scope.Set("_last_expr_result", record)
			scope.Set(strings.ToLower(modelName), record)
			return nil
		}

		// Model::where($col, $val) or Model::where($col, $op, $val)
		if methodName == "where" {
			qb := orm.Query(modelName)
			if len(node.Children) == 2 {
				col := fmt.Sprintf("%v", node.Children[0].Value)
				val := node.Children[1].Value
				qb.Where(col, val)
			} else if len(node.Children) >= 3 {
				col := fmt.Sprintf("%v", node.Children[0].Value)
				op := fmt.Sprintf("%v", node.Children[1].Value)
				val := node.Children[2].Value
				qb.Where(col, op, val)
			}

			// Chained calls like ->get(), ->first(), ->orderBy()
			records, err := qb.Get()
			if err != nil {
				scope.Set("_last_expr_result", []interface{}{})
			} else {
				scope.Set("_last_expr_result", records)
			}
			return nil
		}

		// Model::count()
		if methodName == "count" {
			count, err := orm.Query(modelName).Count()
			if err != nil {
				scope.Set("_last_expr_result", 0)
			} else {
				scope.Set("_last_expr_result", count)
			}
			return nil
		}

		return nil
	})

	// 6. SLOT: laravel.request_call ($request->user(), request()->all())
	eng.Register("laravel.request_call", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		methodName := fmt.Sprintf("%v", node.Value)
		if methodName == "user" {
			if authUser, ok := scope.Get("auth.user"); ok {
				scope.Set("_last_expr_result", authUser)
			} else {
				scope.Set("_last_expr_result", nil)
			}
		}
		return nil
	})

	// 7. SLOT: laravel.obj_call (Instance method seperti $product->delete(), $product->update(), dll)
	eng.Register("laravel.obj_call", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		callTarget := fmt.Sprintf("%v", node.Value)
		parts := strings.Split(callTarget, "->")
		if len(parts) < 2 {
			return nil
		}
		varName := strings.TrimPrefix(parts[0], "$")
		methodName := parts[len(parts)-1]

		if methodName == "delete" {
			if objVal, ok := scope.Get(varName); ok {
				if objMap, isMap := objVal.(map[string]interface{}); isMap {
					targetModel := varName
					if mName, hasMName := objMap["_model_name"].(string); hasMName && mName != "" {
						targetModel = mName
					}
					if id, idOk := objMap["id"]; idOk {
						_ = orm.Delete(targetModel, id)
					}
				}
			}
			return nil
		}

		if methodName == "update" {
			if objVal, ok := scope.Get(varName); ok {
				if objMap, isMap := objVal.(map[string]interface{}); isMap {
					targetModel := varName
					if mName, hasMName := objMap["_model_name"].(string); hasMName && mName != "" {
						targetModel = mName
					}
					if id, idOk := objMap["id"]; idOk {
						reqVal, _ := scope.Get("http.request")
						updateData := make(map[string]interface{})
						if reqVal != nil {
							req := reqVal.(*http.Request)
							if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
								_ = json.NewDecoder(req.Body).Decode(&updateData)
							} else if strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
								_ = req.ParseMultipartForm(32 << 20)
								if req.MultipartForm != nil {
									for k, v := range req.MultipartForm.Value {
										if k != "_token" && k != "_method" && len(v) > 0 {
											updateData[k] = v[0]
										}
									}
									for fileKey, fileHeaders := range req.MultipartForm.File {
										if len(fileHeaders) > 0 {
											fh := fileHeaders[0]
											uploadsDir := filepath.Join(appCtx.ProjectRoot, "public", "uploads")
											_ = os.MkdirAll(uploadsDir, 0755)
											ext := filepath.Ext(fh.Filename)
											saveFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
											destPath := filepath.Join(uploadsDir, saveFilename)

											src, err := fh.Open()
											if err == nil {
												dst, err := os.Create(destPath)
												if err == nil {
													_, _ = io.Copy(dst, src)
													_ = dst.Close()
													updateData[fileKey] = "/uploads/" + saveFilename
												}
												_ = src.Close()
											}
										}
									}
								}
							} else {
								_ = req.ParseForm()
								for k, v := range req.PostForm {
									if k != "_token" && k != "_method" && len(v) > 0 {
										updateData[k] = v[0]
									}
								}
							}
						}
						updated, err := orm.Update(targetModel, id, updateData)
						if err == nil {
							scope.Set("_last_expr_result", updated)
							scope.Set(varName, updated)
						}
					}
				}
			}
			return nil
		}

		// $request->validate([...])
		if methodName == "validate" {
			reqVal, _ := scope.Get("http.request")
			wVal, _ := scope.Get("http.response_writer")
			if reqVal != nil && wVal != nil {
				req := reqVal.(*http.Request)
				w := wVal.(http.ResponseWriter)

				// Parse input data
				inputData := make(map[string]interface{})
				if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
					if req.Body != nil {
						bodyBytes, _ := io.ReadAll(req.Body)
						req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
						_ = json.Unmarshal(bodyBytes, &inputData)
					}
				} else {
					_ = req.ParseForm()
					for k, v := range req.Form {
						if len(v) > 0 {
							inputData[k] = v[0]
						}
					}
				}

				// Check validation rules from node.Children
				validationErrors := make(map[string][]string)
				for _, child := range node.Children {
					field := fmt.Sprintf("%v", child.Name)
					rule := fmt.Sprintf("%v", child.Value)
					val, exists := inputData[field]

					rules := strings.Split(rule, "|")
					for _, r := range rules {
						r = strings.TrimSpace(r)
						if r == "required" {
							if !exists || val == nil || fmt.Sprintf("%v", val) == "" {
								validationErrors[field] = append(validationErrors[field], fmt.Sprintf("The %s field is required.", field))
							}
						} else if strings.HasPrefix(r, "min:") {
							minStr := strings.TrimPrefix(r, "min:")
							minLen := 0
							_, _ = fmt.Sscanf(minStr, "%d", &minLen)
							if exists && len(fmt.Sprintf("%v", val)) < minLen {
								validationErrors[field] = append(validationErrors[field], fmt.Sprintf("The %s field must be at least %d characters.", field, minLen))
							}
						}
					}
				}

				if len(validationErrors) > 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnprocessableEntity) // 422 Unprocessable Content
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"message": "The given data was invalid.",
						"errors":  validationErrors,
					})
					scope.Set("_response_sent", true)
					return fmt.Errorf("validation failed")
				}

				scope.Set("_last_expr_result", inputData)
			}
			return nil
		}

		// Generic Authentication Service Dispatcher
		if methodName == "login" {
			reqVal, _ := scope.Get("http.request")
			var email, password string
			if reqVal != nil {
				req := reqVal.(*http.Request)
				if req.Method == "POST" {
					if req.Body != nil {
						bodyBytes, _ := io.ReadAll(req.Body)
						req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
						var payload map[string]interface{}
						if err := json.Unmarshal(bodyBytes, &payload); err == nil {
							if em, ok := payload["email"].(string); ok {
								email = em
							}
							if pw, ok := payload["password"].(string); ok {
								password = pw
							}
						}
					}
					if email == "" {
						email = req.FormValue("email")
						password = req.FormValue("password")
					}
				}
			}

			// Prioritaskan verifikasi via configured API backend jika ada
			var backendURL string
			if appCtx.Env != nil {
				for k, v := range appCtx.Env {
					if (strings.HasSuffix(k, "_BACKEND_URL") || strings.HasSuffix(k, "_API_URL") || k == "BACKEND_URL" || k == "API_URL") && v != "" {
						backendURL = strings.TrimRight(v, "/")
						break
					}
				}
			}

			var authResult map[string]interface{}
			if backendURL != "" {
				loginBody, _ := json.Marshal(map[string]string{
					"email":    email,
					"password": password,
				})
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Post(backendURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(loginBody))
				if err != nil {
					return fmt.Errorf("Layanan autentikasi backend tidak dapat dihubungi: %w", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode >= 400 {
					var errResp map[string]interface{}
					_ = json.NewDecoder(resp.Body).Decode(&errResp)
					msg := "Email atau password yang Anda masukkan salah."
					if m, ok := errResp["message"].(string); ok && m != "" {
						msg = m
					}
					return fmt.Errorf("%s", msg)
				}

				var apiResp struct {
					Data map[string]interface{} `json:"data"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&apiResp); err == nil && apiResp.Data != nil {
					authResult = apiResp.Data
				}
			} else {
				// Fallback verifikasi lokal via ORM database
				user, token, err := orm.VerifyCredentials("User", email, password, "web-token")
				if err != nil {
					return fmt.Errorf("Email atau password yang Anda masukkan salah.")
				}
				authResult = map[string]interface{}{
					"user": user,
					"tokens": map[string]interface{}{
						"access_token":  token,
						"refresh_token": token,
					},
				}
			}

			scope.Set("_last_expr_result", authResult)
			return nil
		}

		return nil
	})

	// 8. SLOT: php.assign ($x = ...)
	eng.Register("php.assign", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		varName := strings.TrimPrefix(fmt.Sprintf("%v", node.Value), "$")
		for _, c := range node.Children {
			if err := eng.Execute(ctx, c, scope); err != nil {
				return err
			}
		}
		if res, ok := scope.Get("_last_expr_result"); ok {
			scope.Set(varName, res)
		}
		return nil
	})

	// 9. SLOT: laravel.response_json (response()->json(...))
	eng.Register("laravel.response_json", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		wVal, ok := scope.Get("http.response_writer")
		if !ok {
			return fmt.Errorf("http.response_writer not found")
		}
		w := wVal.(http.ResponseWriter)

		var outputData interface{} = nil
		statusCode := http.StatusOK

		if len(node.Children) > 0 {
			firstChild := node.Children[0]
			if firstChild.Name == "laravel.model_call" {
				if err := eng.Execute(ctx, firstChild, scope); err == nil {
					if res, found := scope.Get("_last_expr_result"); found {
						outputData = res
					}
				}
			} else if len(node.Children) == 1 {
				rawVal := fmt.Sprintf("%v", firstChild.Value)
				if strings.HasPrefix(rawVal, "$") {
					varName := strings.TrimPrefix(rawVal, "$")
					if val, found := scope.Get(varName); found {
						outputData = val
					}
				} else {
					outputData = rawVal
				}
			} else if len(node.Children) == 2 && strings.HasPrefix(fmt.Sprintf("%v", firstChild.Value), "$") {
				// Format: response()->json($var, 201)
				varName := strings.TrimPrefix(fmt.Sprintf("%v", firstChild.Value), "$")
				if val, found := scope.Get(varName); found {
					outputData = val
				}
				secondVal := fmt.Sprintf("%v", node.Children[1].Value)
				if secondVal == "201" {
					statusCode = http.StatusCreated
				} else if secondVal == "404" {
					statusCode = http.StatusNotFound
				} else if secondVal == "401" {
					statusCode = http.StatusUnauthorized
				}
			} else {
				dataMap := make(map[string]interface{})
				items := node.Children
				for i := 0; i < len(items); i++ {
					valStr := fmt.Sprintf("%v", items[i].Value)
					if valStr == "201" {
						statusCode = http.StatusCreated
						continue
					} else if valStr == "404" {
						statusCode = http.StatusNotFound
						continue
					} else if valStr == "401" {
						statusCode = http.StatusUnauthorized
						continue
					}

					if i+1 < len(items) {
						key := valStr
						valRaw := fmt.Sprintf("%v", items[i+1].Value)
						if valRaw == "201" || valRaw == "404" || valRaw == "401" {
							continue
						}
						var finalVal interface{} = valRaw
						if valRaw == "$request->user()" || valRaw == "request()->user()" {
							if authUser, found := scope.Get("auth.user"); found {
								finalVal = authUser
							}
						} else if strings.HasPrefix(valRaw, "$") {
							if v, found := scope.Get(strings.TrimPrefix(valRaw, "$")); found {
								finalVal = v
							}
						}
						dataMap[key] = finalVal
						i++
					}
				}
				if len(dataMap) > 0 {
					outputData = dataMap
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		return json.NewEncoder(w).Encode(outputData)
	})

	// 10. SLOT: laravel.view (Menggantikan return view('products.index', [...]))
	eng.Register("laravel.view", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		wVal, ok := scope.Get("http.response_writer")
		if !ok {
			return fmt.Errorf("http.response_writer not found")
		}
		w := wVal.(http.ResponseWriter)

		viewName := fmt.Sprintf("%v", node.Value)
		data := make(map[string]interface{})

		// Ambil data yang dikirim ke view
		data["appName"] = filepath.Base(appCtx.ProjectRoot)
		for _, c := range node.Children {
			rawVal := fmt.Sprintf("%v", c.Value)
			if strings.HasPrefix(rawVal, "$") {
				varName := strings.TrimPrefix(rawVal, "$")
				if val, found := scope.Get(varName); found {
					data[varName] = val
				}
			}
		}

		bladeEng := blade.NewBladeEngine(appCtx.ProjectRoot)
		htmlOutput, err := bladeEng.Render(viewName, data)
		if err != nil {
			http.Error(w, fmt.Sprintf("Blade Render Error: %v", err), 500)
			return err
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(htmlOutput))
		return err
	})

	// 11. SLOT: laravel.redirect (Menggantikan return redirect(...))
	eng.Register("laravel.redirect", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		wVal, ok := scope.Get("http.response_writer")
		if !ok {
			return fmt.Errorf("http.response_writer not found")
		}
		w := wVal.(http.ResponseWriter)
		rVal, _ := scope.Get("http.request")
		var r *http.Request
		if rVal != nil {
			r = rVal.(*http.Request)
		}

		targetURL := fmt.Sprintf("%v", node.Value)
		// Resolve jika targetURL adalah route name (misal dashboard.index -> /dashboard)
		if resolvedPath, exists := appCtx.DynamicRouter.NamedRoutes[targetURL]; exists {
			targetURL = resolvedPath
		}
		if targetURL == "back" && r != nil {
			targetURL = r.Header.Get("Referer")
			if targetURL == "" {
				targetURL = "/"
			}
		} else if targetURL == "" || targetURL == "intended" {
			targetURL = "/"
		}
		if !strings.HasPrefix(targetURL, "/") && !strings.HasPrefix(targetURL, "http") {
			targetURL = "/" + targetURL
		}

		scope.Set("_response_sent", true)
		if r != nil && r.Header.Get("X-Inertia") == "true" {
			http.Redirect(w, r, targetURL, http.StatusSeeOther)
		} else if r != nil {
			http.Redirect(w, r, targetURL, http.StatusFound)
		}
		return nil
	})

	return nil
}
