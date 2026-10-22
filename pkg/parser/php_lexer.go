package parser

import (
	"unicode"
)

type TokenType string

const (
	TokenEOF        TokenType = "EOF"
	TokenIdentifier TokenType = "IDENTIFIER"
	TokenVariable   TokenType = "VARIABLE"   // e.g. $id, $request
	TokenString     TokenType = "STRING"     // e.g. 'users', "admin"
	TokenNumber     TokenType = "NUMBER"     // e.g. 1, 404
	TokenDoubleCol  TokenType = "DOUBLE_COL" // ::
	TokenArrow      TokenType = "ARROW"      // ->
	TokenLParen     TokenType = "LPAREN"     // (
	TokenRParen     TokenType = "RPAREN"     // )
	TokenLBrace     TokenType = "LBRACE"     // {
	TokenRBrace     TokenType = "RBRACE"     // }
	TokenLBracket   TokenType = "LBRACKET"   // [
	TokenRBracket   TokenType = "RBRACKET"   // ]
	TokenComma      TokenType = "COMMA"      // ,
	TokenSemicolon  TokenType = "SEMICOLON"  // ;
	TokenAssign     TokenType = "ASSIGN"     // =
	TokenFatArrow   TokenType = "FAT_ARROW"  // =>
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

type PHPLexer struct {
	input   string
	pos     int
	readPos int
	ch      byte
	line    int
	col     int
}

func NewPHPLexer(input string) *PHPLexer {
	l := &PHPLexer{input: input, line: 1, col: 0}
	l.readChar()
	return l
}

func (l *PHPLexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *PHPLexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *PHPLexer) skipWhitespaceAndComments() {
	for {
		// Whitespace
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
			continue
		}
		if l.ch == '\n' {
			l.line++
			l.col = 0
			l.readChar()
			continue
		}

		// Comment // or /* */
		if l.ch == '/' {
			if l.peekChar() == '/' {
				for l.ch != '\n' && l.ch != 0 {
					l.readChar()
				}
				continue
			} else if l.peekChar() == '*' {
				l.readChar()
				l.readChar()
				for !(l.ch == '*' && l.peekChar() == '/') && l.ch != 0 {
					if l.ch == '\n' {
						l.line++
						l.col = 0
					}
					l.readChar()
				}
				if l.ch != 0 {
					l.readChar()
					l.readChar()
				}
				continue
			}
		}

		// PHP 8 Attribute #[...] or single line comment #
		if l.ch == '#' {
			if l.peekChar() == '[' {
				// Skip #[...] attribute
				l.readChar()
				l.readChar()
				depth := 1
				for depth > 0 && l.ch != 0 {
					if l.ch == '[' {
						depth++
					} else if l.ch == ']' {
						depth--
					} else if l.ch == '\n' {
						l.line++
						l.col = 0
					}
					l.readChar()
				}
				continue
			} else {
				// Single line comment #
				for l.ch != '\n' && l.ch != 0 {
					l.readChar()
				}
				continue
			}
		}

		// PHP tag <?php
		if l.ch == '<' && l.peekChar() == '?' {
			for l.ch != '\n' && l.ch != ' ' && l.ch != '\t' && l.ch != 0 {
				l.readChar()
			}
			continue
		}

		break
	}
}

func (l *PHPLexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	var tok Token
	tok.Line = l.line
	tok.Col = l.col

	switch l.ch {
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenDoubleCol, Literal: string(ch) + string(l.ch), Line: l.line, Col: l.col}
		} else {
			tok = Token{Type: TokenIdentifier, Literal: ":", Line: l.line, Col: l.col}
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenArrow, Literal: string(ch) + string(l.ch), Line: l.line, Col: l.col}
		} else {
			tok = Token{Type: TokenIdentifier, Literal: "-", Line: l.line, Col: l.col}
		}
	case '=':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenFatArrow, Literal: string(ch) + string(l.ch), Line: l.line, Col: l.col}
		} else {
			tok = Token{Type: TokenAssign, Literal: "=", Line: l.line, Col: l.col}
		}
	case '(':
		tok = Token{Type: TokenLParen, Literal: "(", Line: l.line, Col: l.col}
	case ')':
		tok = Token{Type: TokenRParen, Literal: ")", Line: l.line, Col: l.col}
	case '{':
		tok = Token{Type: TokenLBrace, Literal: "{", Line: l.line, Col: l.col}
	case '}':
		tok = Token{Type: TokenRBrace, Literal: "}", Line: l.line, Col: l.col}
	case '[':
		tok = Token{Type: TokenLBracket, Literal: "[", Line: l.line, Col: l.col}
	case ']':
		tok = Token{Type: TokenRBracket, Literal: "]", Line: l.line, Col: l.col}
	case ',':
		tok = Token{Type: TokenComma, Literal: ",", Line: l.line, Col: l.col}
	case ';':
		tok = Token{Type: TokenSemicolon, Literal: ";", Line: l.line, Col: l.col}
	case '"', '\'':
		tok.Type = TokenString
		tok.Literal = l.readString(l.ch)
		return tok
	case '$':
		tok.Type = TokenVariable
		l.readChar()
		tok.Literal = "$" + l.readIdentifier()
		return tok
	case 0:
		tok.Type = TokenEOF
		tok.Literal = ""
		return tok
	default:
		if isLetter(l.ch) || l.ch == '\\' {
			tok.Type = TokenIdentifier
			tok.Literal = l.readIdentifier()
			return tok
		} else if isDigit(l.ch) {
			tok.Type = TokenNumber
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = Token{Type: TokenIdentifier, Literal: string(l.ch), Line: l.line, Col: l.col}
		}
	}

	l.readChar()
	return tok
}

func (l *PHPLexer) readString(quote byte) string {
	l.readChar()
	start := l.pos
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	lit := l.input[start:l.pos]
	if l.ch == quote {
		l.readChar()
	}
	return lit
}

func (l *PHPLexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '\\' {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *PHPLexer) readNumber() string {
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}
