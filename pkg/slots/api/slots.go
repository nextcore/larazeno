package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"larazeno-core/pkg/engine"
)

// RegisterAPISlots mendaftarkan slots khusus kebutuhan REST API Laravel
func RegisterAPISlots(eng *engine.Engine) {

	// 1. SLOT: auth.token_guard (Mencegah kerentanan GHSA-f9x3-7vq9-cph6 secara strict di Go)
	eng.Register("auth.token_guard", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		reqVal, ok := scope.Get("http.request")
		if !ok {
			return fmt.Errorf("http.request not found in scope")
		}
		req := reqVal.(*http.Request)

		// Ekstraksi parameter token
		paramName := "api_token"
		if node.Value != nil && fmt.Sprintf("%v", node.Value) != "" {
			paramName = fmt.Sprintf("%v", node.Value)
		}

		// Pengecekan tipe data secara ketat di Go:
		// Jika client mengirim array (misal: api_token[]), query param akan memiliki key 'api_token[]'
		for qKey := range req.URL.Query() {
			if strings.HasPrefix(qKey, paramName+"[") || qKey == paramName+"[]" {
				// Deteksi input array -> tolak langsung!
				scope.Set("auth.status", "unauthenticated")
				scope.Set("auth.error", "Array type token rejected (Strict Type Safety).")
				return executeChild(ctx, eng, node, "unauthorized", scope)
			}
		}

		token := req.URL.Query().Get(paramName)
		if token == "" {
			// Cek Authorization Bearer Header
			authHeader := req.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			scope.Set("auth.status", "unauthenticated")
			return executeChild(ctx, eng, node, "unauthorized", scope)
		}

		// Simulasi lookup user autentikasi
		if token == "secret123" {
			scope.Set("auth.user", map[string]interface{}{
				"id":    1,
				"name":  "Admin Go Engine",
				"email": "admin@engine.local",
			})
			return executeChild(ctx, eng, node, "authorized", scope)
		}

		scope.Set("auth.status", "invalid_token")
		return executeChild(ctx, eng, node, "unauthorized", scope)
	})

	// 2. SLOT: model.find (Simulasi Eloquent User::find($id))
	eng.Register("model.find", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		targetVar := "result"
		for _, c := range node.Children {
			if c.Name == "as" && c.Value != nil {
				targetVar = fmt.Sprintf("%v", c.Value)
			}
		}

		idVal := "1"
		if node.Value != nil {
			idVal = fmt.Sprintf("%v", node.Value)
		}

		// Mock Data Eloquent Record
		record := map[string]interface{}{
			"id":         idVal,
			"name":       "John Doe",
			"email":      "john@laravel.local",
			"created_at": "2026-10-21 14:00:00",
		}

		scope.Set(targetVar, record)
		return nil
	})

	// 3. SLOT: response.json (Mengembalikan JSON response HTTP)
	eng.Register("response.json", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		wVal, ok := scope.Get("http.response_writer")
		if !ok {
			return fmt.Errorf("http.response_writer not found in scope")
		}
		w := wVal.(http.ResponseWriter)

		statusCode := 200
		var payload interface{} = nil

		for _, c := range node.Children {
			if c.Name == "status" && c.Value != nil {
				fmt.Sscanf(fmt.Sprintf("%v", c.Value), "%d", &statusCode)
			}
			if c.Name == "data" {
				valStr := fmt.Sprintf("%v", c.Value)
				if strings.HasPrefix(valStr, "$") {
					varName := strings.TrimPrefix(valStr, "$")
					if val, found := scope.Get(varName); found {
						payload = val
					}
				} else {
					payload = c.Value
				}
			}
			if c.Name == "message" && c.Value != nil {
				payload = map[string]interface{}{
					"message": c.Value,
				}
			}
		}

		if payload == nil && node.Value != nil {
			valStr := fmt.Sprintf("%v", node.Value)
			if strings.HasPrefix(valStr, "$") {
				varName := strings.TrimPrefix(valStr, "$")
				if val, found := scope.Get(varName); found {
					payload = val
				}
			} else {
				payload = node.Value
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		return json.NewEncoder(w).Encode(payload)
	})
}

func executeChild(ctx context.Context, eng *engine.Engine, node *engine.Node, childName string, scope *engine.Scope) error {
	for _, c := range node.Children {
		if c.Name == childName {
			for _, sub := range c.Children {
				if err := eng.Execute(ctx, sub, scope); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return nil
}
