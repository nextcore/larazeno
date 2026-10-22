package parser

import (
	"fmt"
	"testing"
)

func TestProductControllerParser(t *testing.T) {
	node, err := ParsePHPFile("/media/max/3C38C268119D6B5C/projects/experimental/laravel-sanctum-v13/app/Http/Controllers/ProductController.php")
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	for _, c := range node.Children {
		fmt.Printf("Class: %v (%s)\n", c.Value, c.Name)
	}
}

func TestAuthParser(t *testing.T) {
	node, err := ParsePHPFile("/media/max/3C38C268119D6B5C/projects/experimental/web-kota-admin/app/Http/Controllers/Auth/AuthenticatedSessionController.php")
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	for _, c := range node.Children {
		fmt.Printf("Class: %v (%s)\n", c.Value, c.Name)
		for _, m := range c.Children {
			fmt.Printf("  Method: %v (%s), children=%d\n", m.Value, m.Name, len(m.Children))
			for _, s := range m.Children {
				fmt.Printf("    Stmt: %s val=%v (children=%d)\n", s.Name, s.Value, len(s.Children))
				for _, sub := range s.Children {
					fmt.Printf("      Sub: %s val=%v (children=%d)\n", sub.Name, sub.Value, len(sub.Children))
					for _, ssub := range sub.Children {
						fmt.Printf("        SubSub: %s val=%v (children=%d)\n", ssub.Name, ssub.Value, len(ssub.Children))
					}
				}
			}

		}
	}
}

