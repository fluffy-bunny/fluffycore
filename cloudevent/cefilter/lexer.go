package cefilter

import (
	"fmt"
	"strings"
	"unicode"
)

// -------------------------------------------------------------------------
// Token types
// -------------------------------------------------------------------------

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent           // type  source  orgid  ...
	tokString          // "hello"  'hello'
	tokEq              // =
	tokNe              // !=
	tokLike            // LIKE
	tokIn              // IN
	tokExists          // EXISTS
	tokAnd             // AND
	tokOr              // OR
	tokNot             // NOT
	tokLParen          // (
	tokRParen          // )
	tokComma           // ,
)

type token struct {
	kind tokenKind
	val  string // meaningful for tokIdent and tokString
	pos  int    // byte offset in input, for error messages
}

func (t token) String() string {
	switch t.kind {
	case tokEOF:
		return "<EOF>"
	case tokIdent:
		return fmt.Sprintf("IDENT(%s)", t.val)
	case tokString:
		return fmt.Sprintf("STRING(%q)", t.val)
	default:
		return t.val
	}
}

// -------------------------------------------------------------------------
// Lexer
// -------------------------------------------------------------------------

type lexer struct {
	input []rune
	pos   int
	toks  []token
	err   error
}

func lex(input string) ([]token, error) {
	l := &lexer{input: []rune(input)}
	l.tokenize()
	if l.err != nil {
		return nil, l.err
	}
	return l.toks, nil
}

func (l *lexer) tokenize() {
	for {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			l.emit(tokEOF, "", l.pos)
			return
		}
		start := l.pos
		ch := l.input[l.pos]

		switch {
		case ch == '(':
			l.pos++
			l.emit(tokLParen, "(", start)
		case ch == ')':
			l.pos++
			l.emit(tokRParen, ")", start)
		case ch == ',':
			l.pos++
			l.emit(tokComma, ",", start)
		case ch == '!' && l.peek(1) == '=':
			l.pos += 2
			l.emit(tokNe, "!=", start)
		case ch == '=':
			l.pos++
			l.emit(tokEq, "=", start)
		case ch == '"' || ch == '\'':
			s, err := l.readString(ch)
			if err != nil {
				l.err = err
				return
			}
			l.emit(tokString, s, start)
		case unicode.IsLetter(ch) || ch == '_':
			word := l.readWord()
			l.emitWord(word, start)
		default:
			l.err = fmt.Errorf("unexpected character %q at position %d", string(ch), start)
			return
		}
	}
}

func (l *lexer) peek(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.input) {
		return 0
	}
	return l.input[idx]
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) readWord() string {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' || ch == '.' {
			l.pos++
		} else {
			break
		}
	}
	return string(l.input[start:l.pos])
}

func (l *lexer) readString(quote rune) (string, error) {
	l.pos++ // consume opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == quote {
			l.pos++ // consume closing quote
			return sb.String(), nil
		}
		if ch == '\\' && l.pos+1 < len(l.input) {
			l.pos++
			sb.WriteRune(l.input[l.pos])
			l.pos++
			continue
		}
		sb.WriteRune(ch)
		l.pos++
	}
	return "", fmt.Errorf("unterminated string starting at position %d", l.pos)
}

func (l *lexer) emit(kind tokenKind, val string, pos int) {
	l.toks = append(l.toks, token{kind: kind, val: val, pos: pos})
}

// emitWord resolves reserved words (case-insensitive) from plain identifiers.
func (l *lexer) emitWord(word string, pos int) {
	switch strings.ToUpper(word) {
	case "AND":
		l.emit(tokAnd, "AND", pos)
	case "OR":
		l.emit(tokOr, "OR", pos)
	case "NOT":
		l.emit(tokNot, "NOT", pos)
	case "LIKE":
		l.emit(tokLike, "LIKE", pos)
	case "IN":
		l.emit(tokIn, "IN", pos)
	case "EXISTS":
		l.emit(tokExists, "EXISTS", pos)
	default:
		l.emit(tokIdent, word, pos)
	}
}
