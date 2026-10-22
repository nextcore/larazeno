package blade

import (
	"fmt"
	"testing"
)

func TestBladeRendering(t *testing.T) {
	engine := NewBladeEngine("/media/max/3C38C268119D6B5C/projects/experimental/laravel-sanctum-v13")
	data := map[string]interface{}{
		"appName": "LaraZeno Modern Store",
		"products": []map[string]interface{}{
			{"name": "UltraWide Monitor", "description": "165Hz Curved Panel", "price": 499.5},
			{"name": "Keychron Q1 Max", "description": "Mechanical keyboard", "price": 219.0},
		},
	}
	out, err := engine.Render("catalog", data)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	fmt.Println("Rendered HTML:")
	fmt.Println(out)
}
