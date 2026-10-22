package parser

import (
	"fmt"
	"testing"
)

func TestLexer(t *testing.T) {
	code := `<?php
namespace App\Http\Controllers;

class UserController {
    public function show($id) {
        $user = User::find($id);
        return response()->json($user);
    }
}`
	lexer := NewPHPLexer(code)
	for {
		tok := lexer.NextToken()
		fmt.Printf("TOKEN: %+v\n", tok)
		if tok.Type == TokenEOF {
			break
		}
	}
}
