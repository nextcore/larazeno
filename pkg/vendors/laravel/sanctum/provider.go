package sanctum

import (
	"context"
	"fmt"
	"strings"

	"larazeno-core/pkg/engine"
	"larazeno-core/pkg/slots/sanctum"
	"larazeno-core/pkg/vendors"
)

type SanctumProvider struct{}

func (s *SanctumProvider) PackageName() string {
	return "laravel/sanctum"
}

func init() {
	vendors.Register(&SanctumProvider{})
}

func (s *SanctumProvider) RegisterSlots(eng *engine.Engine, appCtx *vendors.AppContext) error {

	// 1. SLOT: laravel.sanctum_create_token (User::createToken / $user->createToken)
	eng.Register("laravel.obj_call", func(ctx context.Context, node *engine.Node, scope *engine.Scope) error {
		callTarget := fmt.Sprintf("%v", node.Value)
		parts := strings.Split(callTarget, "->")
		if len(parts) < 2 {
			return nil
		}
		varName := strings.TrimPrefix(parts[0], "$")
		methodName := parts[1]

		if methodName == "createToken" {
			tokenName := "api-token"
			if len(node.Children) > 0 {
				tokenName = fmt.Sprintf("%v", node.Children[0].Value)
			}
			var userID int64 = 1
			modelClass := "App\\Models\\User"
			if uVal, ok := scope.Get(varName); ok {
				if uMap, isMap := uVal.(map[string]interface{}); isMap {
					if id, idOk := uMap["id"]; idOk {
						switch v := id.(type) {
						case int64:
							userID = v
						case int:
							userID = int64(v)
						case float64:
							userID = int64(v)
						}
					}
				}
			}
			newTok, err := sanctum.CreateToken(modelClass, userID, tokenName, []string{"*"})
			if err != nil {
				return err
			}
			scope.Set("_last_expr_result", newTok.PlainTextToken)
			scope.Set("token", newTok.PlainTextToken)
			return nil
		}

		return nil
	})

	return nil
}
