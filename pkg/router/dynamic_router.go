package router

import (
	"regexp"
	"strings"

	"larazeno-core/pkg/engine"
)

type RouteEntry struct {
	Method      string
	Pattern     string
	Regex       *regexp.Regexp
	ParamNames  []string
	Action      string // e.g. "ProductController@index"
	Middleware  string // e.g. "auth:sanctum"
	HandlerNode *engine.Node
}

type DynamicRouter struct {
	Routes      []*RouteEntry
	NamedRoutes map[string]string // "dashboard.index" -> "/dashboard"
}

func NewDynamicRouter() *DynamicRouter {
	return &DynamicRouter{
		Routes:      make([]*RouteEntry, 0),
		NamedRoutes: make(map[string]string),
	}
}

// AddRoute converts Laravel route path like "/products/{id}" or "products/{id}" to regexp
func (r *DynamicRouter) AddRoute(method string, path string, action string, middleware string, node *engine.Node) {
	// Standardize path with leading slash
	cleanPath := path
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	// Capture route parameter names: {id}, {slug}, etc.
	reParam := regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)
	matches := reParam.FindAllStringSubmatch(cleanPath, -1)
	var paramNames []string
	for _, m := range matches {
		if len(m) > 1 {
			paramNames = append(paramNames, m[1])
		}
	}

	// Replace {param} with regex capture group ([^/]+)
	patternStr := "^" + reParam.ReplaceAllString(cleanPath, "([^/]+)") + "$"
	compiledRegex, err := regexp.Compile(patternStr)
	if err != nil {
		return
	}

	entry := &RouteEntry{
		Method:      strings.ToUpper(method),
		Pattern:     cleanPath,
		Regex:       compiledRegex,
		ParamNames:  paramNames,
		Action:      action,
		Middleware:  middleware,
		HandlerNode: node,
	}
	r.Routes = append(r.Routes, entry)
}

func (r *DynamicRouter) Match(method string, path string) (*RouteEntry, map[string]string) {
	cleanPath := path
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	cleanPath = strings.TrimSuffix(cleanPath, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}

	upperMethod := strings.ToUpper(method)

	for _, rt := range r.Routes {
		if rt.Method != upperMethod {
			continue
		}

		// Also check with/without /api prefix compatibility
		candidates := []string{cleanPath}
		if strings.HasPrefix(cleanPath, "/api/") {
			candidates = append(candidates, strings.TrimPrefix(cleanPath, "/api"))
		} else {
			candidates = append(candidates, "/api"+cleanPath)
		}

		for _, cand := range candidates {
			submatches := rt.Regex.FindStringSubmatch(cand)
			if len(submatches) > 0 {
				params := make(map[string]string)
				for idx, val := range submatches[1:] {
					if idx < len(rt.ParamNames) {
						params[rt.ParamNames[idx]] = val
					}
				}
				return rt, params
			}
		}
	}
	return nil, nil
}
