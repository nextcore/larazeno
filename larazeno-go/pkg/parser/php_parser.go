package parser

import (
	"fmt"
	"os"
	"strings"

	"larazeno-core/pkg/engine"
)

type PHPParser struct {
	lexer   *PHPLexer
	currTok Token
	peekTok Token
}

func NewPHPParser(input string) *PHPParser {
	p := &PHPParser{lexer: NewPHPLexer(input)}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *PHPParser) nextToken() {
	p.currTok = p.peekTok
	p.peekTok = p.lexer.NextToken()
}

func (p *PHPParser) Parse() (*engine.Node, error) {
	root := &engine.Node{Name: "root"}

	for p.currTok.Type != TokenEOF {
		node := p.parseStatement()
		if node != nil {
			node.Parent = root
			root.Children = append(root.Children, node)
		} else {
			p.nextToken()
		}
	}

	return root, nil
}

func (p *PHPParser) parseStatement() *engine.Node {
	// Skip namespace, use statements
	if p.currTok.Literal == "namespace" || p.currTok.Literal == "use" {
		for p.currTok.Type != TokenSemicolon && p.currTok.Type != TokenEOF {
			p.nextToken()
		}
		if p.currTok.Type == TokenSemicolon {
			p.nextToken() // consume ';'
		}
		return p.parseStatement()
	}

	// 1. CLASS DEFINITION
	if p.currTok.Literal == "class" {
		p.nextToken() // consume 'class'
		className := p.currTok.Literal
		p.nextToken() // consume className

		classNode := &engine.Node{
			Name:  "php.class",
			Value: className,
		}

		for p.currTok.Type != TokenLBrace && p.currTok.Type != TokenEOF {
			p.nextToken()
		}

		if p.currTok.Type == TokenLBrace {
			p.nextToken() // consume '{'
			for p.currTok.Type != TokenRBrace && p.currTok.Type != TokenEOF {
				method := p.parseStatement()
				if method != nil {
					method.Parent = classNode
					classNode.Children = append(classNode.Children, method)
				} else {
					p.nextToken()
				}
			}
			p.nextToken() // consume '}'
		}
		return classNode
	}

	// 2. METHOD / FUNCTION DEFINITION
	if p.currTok.Literal == "public" || p.currTok.Literal == "protected" || p.currTok.Literal == "private" || p.currTok.Literal == "function" {
		for p.currTok.Literal != "function" && p.currTok.Type != TokenEOF {
			p.nextToken()
		}
		if p.currTok.Literal == "function" {
			p.nextToken() // consume 'function'
			funcName := p.currTok.Literal
			p.nextToken() // consume funcName

			funcNode := &engine.Node{
				Name:  "php.method",
				Value: funcName,
			}

			// Parse args (...)
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
					if p.currTok.Type == TokenVariable {
						argNode := &engine.Node{Name: "arg", Value: p.currTok.Literal}
						funcNode.Children = append(funcNode.Children, argNode)
					}
					p.nextToken()
				}
				p.nextToken() // consume ')'
			}

			// Skip PHP return type declaration if present (e.g. : Response or : ?Response)
			for p.currTok.Type != TokenLBrace && p.currTok.Type != TokenSemicolon && p.currTok.Type != TokenEOF {
				p.nextToken()
			}

			// Parse body {...}
			if p.currTok.Type == TokenLBrace {
				p.nextToken()
				for p.currTok.Type != TokenRBrace && p.currTok.Type != TokenEOF {
					stmt := p.parseStatement()
					if stmt != nil {
						stmt.Parent = funcNode
						funcNode.Children = append(funcNode.Children, stmt)
					} else {
						p.nextToken()
					}
				}
				p.nextToken() // consume '}'
			}
			return funcNode
		}
	}

	// 3. STATIC CALL (Route::get, User::find)
	if p.currTok.Type == TokenIdentifier && p.peekTok.Type == TokenDoubleCol {
		className := p.currTok.Literal
		p.nextToken() // consume class
		p.nextToken() // consume '::'
		methodName := p.currTok.Literal
		p.nextToken() // consume method

		// Special mapping for Route::get / Route::post
		if className == "Route" {
			return p.parseRouteCall(methodName)
		}

		// Generic Static Call (e.g. User::find)
		staticNode := &engine.Node{
			Name:  "laravel.model_call",
			Value: fmt.Sprintf("%s::%s", className, methodName),
		}

		// Parse arguments (...)
		if p.currTok.Type == TokenLParen {
			p.nextToken()
			for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
				if p.currTok.Type == TokenVariable || p.currTok.Type == TokenString || p.currTok.Type == TokenNumber {
					staticNode.Children = append(staticNode.Children, &engine.Node{
						Name:  "arg",
						Value: p.currTok.Literal,
					})
				}
				p.nextToken()
			}
			p.nextToken() // consume ')'
		}

		// Parse chained calls like ->get(), ->first(), ->orderBy(...)
		for p.currTok.Type == TokenArrow {
			p.nextToken() // consume '->'
			chainedMethod := p.currTok.Literal
			p.nextToken() // consume method name

			chainedNode := &engine.Node{
				Name:  "chained",
				Value: chainedMethod,
			}
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
					if p.currTok.Type == TokenVariable || p.currTok.Type == TokenString || p.currTok.Type == TokenNumber {
						chainedNode.Children = append(chainedNode.Children, &engine.Node{
							Name:  "arg",
							Value: p.currTok.Literal,
						})
					}
					p.nextToken()
				}
				p.nextToken() // consume ')'
			}
			staticNode.Children = append(staticNode.Children, chainedNode)
		}

		return staticNode
	}

	// 4. VARIABLE ASSIGNMENT ($user = User::find($id);)
	if p.currTok.Type == TokenVariable && p.peekTok.Type == TokenAssign {
		varName := p.currTok.Literal
		p.nextToken() // consume variable
		p.nextToken() // consume '='

		assignNode := &engine.Node{
			Name:  "php.assign",
			Value: varName,
		}

		valStmt := p.parseStatement()
		if valStmt != nil {
			valStmt.Parent = assignNode
			assignNode.Children = append(assignNode.Children, valStmt)
		}

		if p.currTok.Type == TokenSemicolon {
			p.nextToken()
		}
		return assignNode
	}

	// 4A. METHOD CALL ON VARIABLE ($token = $user->createToken(...)->plainTextToken or $this->authService->login(...))
	if p.currTok.Type == TokenVariable && p.peekTok.Type == TokenArrow {
		chainParts := []string{p.currTok.Literal}
		p.nextToken() // consume variable

		var callArgs []*engine.Node

		for p.currTok.Type == TokenArrow {
			p.nextToken() // consume '->'
			targetName := p.currTok.Literal
			p.nextToken() // consume property or method name
			chainParts = append(chainParts, targetName)

			if p.currTok.Type == TokenLParen {
				callArgs = nil
				depth := 1
				p.nextToken() // consume '('
				for depth > 0 && p.currTok.Type != TokenEOF {
					if p.currTok.Type == TokenLParen {
						depth++
						p.nextToken()
					} else if p.currTok.Type == TokenRParen {
						depth--
						if depth == 0 {
							p.nextToken() // consume ')'
							break
						}
						p.nextToken()
					} else if p.currTok.Type == TokenLBracket {
						// Parse associative array ['name' => 'required', 'price' => 'required']
						p.nextToken() // consume '['
						for p.currTok.Type != TokenRBracket && p.currTok.Type != TokenEOF {
							if p.currTok.Type == TokenString {
								key := p.currTok.Literal
								p.nextToken()
								if p.currTok.Type == TokenFatArrow {
									p.nextToken() // consume '=>'
									val := ""
									if p.currTok.Type == TokenString {
										val = p.currTok.Literal
										p.nextToken()
									}
									callArgs = append(callArgs, &engine.Node{
										Name:  key,
										Value: val,
									})
								}
							} else {
								p.nextToken()
							}
						}
						if p.currTok.Type == TokenRBracket {
							p.nextToken() // consume ']'
						}
					} else if p.currTok.Type == TokenString || p.currTok.Type == TokenNumber || p.currTok.Type == TokenVariable {
						callArgs = append(callArgs, &engine.Node{
							Name:  "arg",
							Value: p.currTok.Literal,
						})
						p.nextToken()
					} else {
						p.nextToken()
					}
				}
			}
		}

		callNode := &engine.Node{
			Name:     "laravel.obj_call",
			Value:    strings.Join(chainParts, "->"),
			Children: callArgs,
		}

		if p.currTok.Type == TokenSemicolon {
			p.nextToken()
		}
		return callNode
	}

	// 4B. HELPER FUNCTION CALL ($user = request()->user();)
	if p.currTok.Literal == "request" && p.peekTok.Type == TokenLParen {
		p.nextToken() // consume 'request'
		p.nextToken() // consume '('
		if p.currTok.Type == TokenRParen {
			p.nextToken() // consume ')'
		}
		if p.currTok.Type == TokenArrow {
			p.nextToken() // consume '->'
			methodName := p.currTok.Literal
			p.nextToken() // consume method name
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				if p.currTok.Type == TokenRParen {
					p.nextToken()
				}
			}
			return &engine.Node{
				Name:  "laravel.request_call",
				Value: methodName,
			}
		}
	}

	// 4C. IF STATEMENT (if (!$product) { ... })
	if p.currTok.Literal == "if" {
		p.nextToken() // consume 'if'
		// Skip condition (...)
		if p.currTok.Type == TokenLParen {
			depth := 1
			p.nextToken()
			for depth > 0 && p.currTok.Type != TokenEOF {
				if p.currTok.Type == TokenLParen {
					depth++
				} else if p.currTok.Type == TokenRParen {
					depth--
				}
				p.nextToken()
			}
		}
		// Skip body {...}
		if p.currTok.Type == TokenLBrace {
			depth := 1
			p.nextToken()
			for depth > 0 && p.currTok.Type != TokenEOF {
				if p.currTok.Type == TokenLBrace {
					depth++
				} else if p.currTok.Type == TokenRBrace {
					depth--
					if depth == 0 {
						p.nextToken() // consume '}'
						break
					}
				}
				p.nextToken()
			}
		}
		return p.parseStatement()
	}

	// 4D. TRY-CATCH BLOCK (try { ... } catch (...) { ... })
	if p.currTok.Literal == "try" {
		p.nextToken() // consume 'try'
		tryNode := &engine.Node{
			Name: "php.try",
		}
		if p.currTok.Type == TokenLBrace {
			p.nextToken() // consume '{'
			for p.currTok.Type != TokenRBrace && p.currTok.Type != TokenEOF {
				stmt := p.parseStatement()
				if stmt != nil {
					tryNode.Children = append(tryNode.Children, stmt)
				} else {
					p.nextToken()
				}
			}
			if p.currTok.Type == TokenRBrace {
				p.nextToken() // consume '}'
			}
		}

		// Optional catch (...) { ... }
		for p.currTok.Literal == "catch" {
			p.nextToken() // consume 'catch'
			catchNode := &engine.Node{
				Name: "php.catch",
			}
			if p.currTok.Type == TokenLParen {
				depth := 1
				p.nextToken()
				for depth > 0 && p.currTok.Type != TokenEOF {
					if p.currTok.Type == TokenLParen {
						depth++
					} else if p.currTok.Type == TokenRParen {
						depth--
						if depth == 0 {
							p.nextToken() // consume ')'
							break
						}
					}
					p.nextToken()
				}
			}
			if p.currTok.Type == TokenLBrace {
				p.nextToken() // consume '{'
				for p.currTok.Type != TokenRBrace && p.currTok.Type != TokenEOF {
					stmt := p.parseStatement()
					if stmt != nil {
						catchNode.Children = append(catchNode.Children, stmt)
					} else {
						p.nextToken()
					}
				}
				if p.currTok.Type == TokenRBrace {
					p.nextToken() // consume '}'
				}
			}
			tryNode.Children = append(tryNode.Children, catchNode)
		}
		return tryNode
	}

	// 5. RETURN STATEMENT (return response()->json(...);)
	if p.currTok.Literal == "return" {
		p.nextToken() // consume 'return'

		// check response()->json(...)
		if p.currTok.Literal == "response" {
			p.nextToken() // consume 'response'
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				p.nextToken() // consume ')'
			}
			if p.currTok.Type == TokenArrow {
				p.nextToken() // consume '->'
				method := p.currTok.Literal
				p.nextToken() // consume 'json'

				respNode := &engine.Node{
					Name:  "laravel.response_" + method,
					Value: nil,
				}

				// parse payload inside json(...)
				if p.currTok.Type == TokenLParen {
					p.nextToken()
					// Check if argument inside is a static call like Product::all()
					if p.currTok.Type == TokenIdentifier && p.peekTok.Type == TokenDoubleCol {
						innerCall := p.parseStatement()
						if innerCall != nil {
							respNode.Children = append(respNode.Children, innerCall)
						}
					} else {
						for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
							if p.currTok.Type == TokenVariable {
								varName := p.currTok.Literal
								if p.peekTok.Type == TokenArrow {
									p.nextToken() // consume variable
									p.nextToken() // consume '->'
									methodName := p.currTok.Literal
									p.nextToken() // consume method name
									if p.currTok.Type == TokenLParen {
										p.nextToken()
										if p.currTok.Type == TokenRParen {
											p.nextToken()
										}
									}
									respNode.Children = append(respNode.Children, &engine.Node{
										Name:  "data_expr",
										Value: fmt.Sprintf("%s->%s()", varName, methodName),
									})
									continue
								}
								respNode.Children = append(respNode.Children, &engine.Node{
									Name:  "data",
									Value: varName,
								})
							} else if p.currTok.Type == TokenString || p.currTok.Type == TokenNumber {
								respNode.Children = append(respNode.Children, &engine.Node{
									Name:  "data",
									Value: p.currTok.Literal,
								})
							}
							p.nextToken()
						}
					}
					if p.currTok.Type == TokenRParen {
						p.nextToken() // consume ')'
					}
				}

				if p.currTok.Type == TokenSemicolon {
					p.nextToken()
				}
				return respNode
			}
		}

		// check return view('view.name', [...]);
		if p.currTok.Literal == "view" {
			p.nextToken() // consume 'view'
			viewNode := &engine.Node{
				Name: "laravel.view",
			}
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				// First arg: view name
				if p.currTok.Type == TokenString {
					viewNode.Value = p.currTok.Literal
					p.nextToken()
				}
				// Optional second arg: data
				for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
					if p.currTok.Type == TokenVariable || p.currTok.Type == TokenString {
						viewNode.Children = append(viewNode.Children, &engine.Node{
							Name:  "data",
							Value: p.currTok.Literal,
						})
					}
					p.nextToken()
				}
				if p.currTok.Type == TokenRParen {
					p.nextToken() // consume ')'
				}
			}
			if p.currTok.Type == TokenSemicolon {
				p.nextToken()
			}
			return viewNode
		}

		// check return Inertia::render('Component', [...]);
		if p.currTok.Literal == "Inertia" && p.peekTok.Type == TokenDoubleCol {
			p.nextToken() // consume 'Inertia'
			p.nextToken() // consume '::'
			p.nextToken() // consume 'render'
			inertiaNode := &engine.Node{
				Name: "inertia.render",
			}
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				if p.currTok.Type == TokenString {
					inertiaNode.Value = p.currTok.Literal
					p.nextToken()
				}
				if p.currTok.Type == TokenComma {
					p.nextToken()
				}
				// Parse props array [...]
				if p.currTok.Type == TokenLBracket {
					p.nextToken()
					for p.currTok.Type != TokenRBracket && p.currTok.Type != TokenEOF {
						if p.currTok.Type == TokenString {
							propKey := p.currTok.Literal
							p.nextToken()
							if p.currTok.Type == TokenFatArrow {
								p.nextToken() // consume '=>'
								// Cek value
								propVal := ""
								if p.currTok.Type == TokenVariable {
									propVal = p.currTok.Literal
								} else if p.currTok.Type == TokenString {
									propVal = p.currTok.Literal
								}
								inertiaNode.Children = append(inertiaNode.Children, &engine.Node{
									Name:  "prop",
									Value: propKey,
									Children: []*engine.Node{
										{
											Name:  "value",
											Value: propVal,
										},
									},
								})
							}
						}
						p.nextToken()
					}
					if p.currTok.Type == TokenRBracket {
						p.nextToken() // consume ']'
					}
				}
				for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
					p.nextToken()
				}
				if p.currTok.Type == TokenRParen {
					p.nextToken() // consume ')'
				}
			}
			if p.currTok.Type == TokenSemicolon {
				p.nextToken()
			}
			return inertiaNode
		}

		// check return redirect()->intended(...) or redirect(...)
		if p.currTok.Literal == "redirect" {
			p.nextToken() // consume 'redirect'
			redirNode := &engine.Node{
				Name: "laravel.redirect",
			}
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				if p.currTok.Type == TokenString {
					redirNode.Value = p.currTok.Literal
					p.nextToken()
				}
				if p.currTok.Type == TokenRParen {
					p.nextToken()
				}
			}

			// Chained methods like ->back(), ->intended(...), or ->route(...)
			for p.currTok.Type == TokenArrow {
				p.nextToken() // consume '->'
				method := p.currTok.Literal
				p.nextToken() // consume method name

				var innerString string
				if p.currTok.Type == TokenLParen {
					depth := 1
					p.nextToken()
					for depth > 0 && p.currTok.Type != TokenEOF {
						if p.currTok.Type == TokenLParen {
							depth++
							p.nextToken()
						} else if p.currTok.Type == TokenRParen {
							depth--
							if depth == 0 {
								p.nextToken() // consume matching ')'
								break
							}
							p.nextToken()
						} else if p.currTok.Type == TokenString {
							innerString = p.currTok.Literal
							p.nextToken()
						} else {
							p.nextToken()
						}
					}
				}

				if method == "back" {
					redirNode.Value = "back"
				} else if method == "route" || method == "intended" {
					if innerString != "" {
						redirNode.Value = innerString
					} else if redirNode.Value == nil || redirNode.Value == "" {
						redirNode.Value = method
					}
				} else if redirNode.Value == nil || redirNode.Value == "" {
					redirNode.Value = method
				}
			}

			if p.currTok.Type == TokenSemicolon {
				p.nextToken()
			}
			return redirNode
		}

		retVal := p.currTok.Literal
		p.nextToken()
		if p.currTok.Type == TokenSemicolon {
			p.nextToken()
		}
		return &engine.Node{Name: "php.return", Value: retVal}
	}

	return nil
}

func (p *PHPParser) parseRouteCall(method string) *engine.Node {
	// 1. ROUTE GROUPS: Route::prefix(...), Route::middleware(...), Route::group(...)
	if method == "middleware" || method == "prefix" || method == "group" {
		var middlewareName string
		var prefixPath string

		currentMethod := method
		for {
			if currentMethod == "middleware" {
				if p.currTok.Type == TokenLParen {
					p.nextToken()
					if p.currTok.Type == TokenString {
						middlewareName = p.currTok.Literal
						p.nextToken()
					}
					for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
						p.nextToken()
					}
					if p.currTok.Type == TokenRParen {
						p.nextToken()
					}
				}
			} else if currentMethod == "prefix" {
				if p.currTok.Type == TokenLParen {
					p.nextToken()
					if p.currTok.Type == TokenString {
						prefixPath = p.currTok.Literal
						p.nextToken()
					}
					for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
						p.nextToken()
					}
					if p.currTok.Type == TokenRParen {
						p.nextToken()
					}
				}
			}

			if p.currTok.Type == TokenArrow {
				p.nextToken() // consume '->'
				nextMethod := p.currTok.Literal
				p.nextToken() // consume next method name
				if nextMethod == "group" {
					break
				}
				currentMethod = nextMethod
			} else {
				break
			}
		}

		groupNode := &engine.Node{
			Name: "laravel.route_group",
		}
		if prefixPath != "" {
			groupNode.Children = append(groupNode.Children, &engine.Node{
				Name:  "prefix",
				Value: prefixPath,
			})
		}
		if middlewareName != "" {
			groupNode.Children = append(groupNode.Children, &engine.Node{
				Name:  "middleware",
				Value: middlewareName,
			})
		}

		// Parse group(function() { ... })
		if p.currTok.Type == TokenLParen {
			p.nextToken() // consume '('
			for p.currTok.Type != TokenLBrace && p.currTok.Type != TokenEOF {
				p.nextToken()
			}
			if p.currTok.Type == TokenLBrace {
				p.nextToken() // consume '{'
				for p.currTok.Type != TokenRBrace && p.currTok.Type != TokenEOF {
					subRoute := p.parseStatement()
					if subRoute != nil {
						if prefixPath != "" {
							subRoute.Children = append(subRoute.Children, &engine.Node{
								Name:  "group_prefix",
								Value: prefixPath,
							})
						}
						if middlewareName != "" {
							subRoute.Children = append(subRoute.Children, &engine.Node{
								Name:  "middleware",
								Value: middlewareName,
							})
						}
						groupNode.Children = append(groupNode.Children, subRoute)
					} else {
						p.nextToken()
					}
				}
				if p.currTok.Type == TokenRBrace {
					p.nextToken() // consume '}'
				}
			}
			for p.currTok.Type != TokenSemicolon && p.currTok.Type != TokenEOF {
				p.nextToken()
			}
			if p.currTok.Type == TokenSemicolon {
				p.nextToken()
			}
		}
		return groupNode
	}

	// Route::apiResource('products', ProductController::class)
	if method == "apiResource" {
		resourceNode := &engine.Node{
			Name:  "laravel.api_resource",
			Value: "API_RESOURCE",
		}

		if p.currTok.Type == TokenLParen {
			p.nextToken()
			if p.currTok.Type == TokenString {
				resourceNode.Children = append(resourceNode.Children, &engine.Node{
					Name:  "resource_name",
					Value: p.currTok.Literal,
				})
				p.nextToken()
			}
			if p.currTok.Type == TokenComma {
				p.nextToken()
			}
			if p.currTok.Type == TokenIdentifier {
				ctrlName := p.currTok.Literal
				p.nextToken()
				if p.currTok.Type == TokenDoubleCol {
					p.nextToken()
					p.nextToken() // consume 'class'
				}
				resourceNode.Children = append(resourceNode.Children, &engine.Node{
					Name:  "controller",
					Value: ctrlName,
				})
			}
			for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
				p.nextToken()
			}
			p.nextToken() // consume ')'
		}
		return resourceNode
	}

	routeNode := &engine.Node{
		Name:  "laravel.route",
		Value: strings.ToUpper(method),
	}

	if p.currTok.Type == TokenLParen {
		p.nextToken()
		// Path: '/user/{id}'
		if p.currTok.Type == TokenString {
			routeNode.Children = append(routeNode.Children, &engine.Node{
				Name:  "path",
				Value: p.currTok.Literal,
			})
			p.nextToken()
		}
		if p.currTok.Type == TokenComma {
			p.nextToken()
		}

		// Action: [UserController::class, 'show'] or DashboardController::class or function() { ... }
		if p.currTok.Type == TokenLBracket {
			p.nextToken()
			var controllerClass string
			var controllerMethod string

			if p.currTok.Type == TokenIdentifier {
				controllerClass = p.currTok.Literal
				p.nextToken()
				if p.currTok.Type == TokenDoubleCol {
					p.nextToken()
					p.nextToken() // consume 'class'
				}
			}
			if p.currTok.Type == TokenComma {
				p.nextToken()
			}
			if p.currTok.Type == TokenString {
				controllerMethod = p.currTok.Literal
				p.nextToken()
			}

			routeNode.Children = append(routeNode.Children, &engine.Node{
				Name:  "action",
				Value: fmt.Sprintf("%s@%s", controllerClass, controllerMethod),
			})

			for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
				p.nextToken()
			}
			p.nextToken() // consume ')'
		} else if p.currTok.Literal == "function" {
			// Closure route: function() { ... }
			p.nextToken() // consume 'function'
			if p.currTok.Type == TokenLParen {
				for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
					p.nextToken()
				}
				if p.currTok.Type == TokenRParen {
					p.nextToken() // consume ')'
				}
			}
			for p.currTok.Type != TokenLBrace && p.currTok.Type != TokenEOF {
				p.nextToken()
			}

			closureActionNode := &engine.Node{
				Name:  "action",
				Value: "Closure@handle",
			}

			if p.currTok.Type == TokenLBrace {
				p.nextToken() // consume '{'
				for p.currTok.Type != TokenRBrace && p.currTok.Type != TokenEOF {
					stmt := p.parseStatement()
					if stmt != nil {
						closureActionNode.Children = append(closureActionNode.Children, stmt)
					} else {
						p.nextToken()
					}
				}
				if p.currTok.Type == TokenRBrace {
					p.nextToken() // consume '}'
				}
			}

			routeNode.Children = append(routeNode.Children, closureActionNode)
			if p.currTok.Type == TokenRParen {
				p.nextToken() // consume ')'
			}
		} else if p.currTok.Type == TokenIdentifier {
			// Invokable Controller: DashboardController::class
			ctrlName := p.currTok.Literal
			p.nextToken()
			if p.currTok.Type == TokenDoubleCol {
				p.nextToken()
				p.nextToken() // consume 'class'
			}
			routeNode.Children = append(routeNode.Children, &engine.Node{
				Name:  "action",
				Value: fmt.Sprintf("%s@__invoke", ctrlName),
			})
			for p.currTok.Type != TokenRParen && p.currTok.Type != TokenEOF {
				p.nextToken()
			}
			p.nextToken() // consume ')'
		}
	}

	// Cek chained method seperti ->middleware('auth:sanctum')
	for p.currTok.Type == TokenArrow {
		p.nextToken() // consume '->'
		methodName := p.currTok.Literal
		p.nextToken() // consume method name

		if methodName == "middleware" {
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				if p.currTok.Type == TokenString {
					routeNode.Children = append(routeNode.Children, &engine.Node{
						Name:  "middleware",
						Value: p.currTok.Literal,
					})
					p.nextToken()
				}
				if p.currTok.Type == TokenRParen {
					p.nextToken()
				}
			}
		} else if methodName == "name" {
			if p.currTok.Type == TokenLParen {
				p.nextToken()
				if p.currTok.Type == TokenString {
					routeNode.Children = append(routeNode.Children, &engine.Node{
						Name:  "route_name",
						Value: p.currTok.Literal,
					})
					p.nextToken()
				}
				if p.currTok.Type == TokenRParen {
					p.nextToken()
				}
			}
		}
	}

	return routeNode
}

func ParsePHPFile(path string) (*engine.Node, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parser := NewPHPParser(string(bytes))
	return parser.Parse()
}
