package cefilter

import (
	"fmt"
	"strings"
)

// Parse compiles a filter expression string into a Predicate tree.
// The returned Predicate is safe to store and reuse across many events;
// it holds no per-call state.
//
// Grammar (precedence low → high):
//
//	expr     = or_expr
//	or_expr  = and_expr  { OR  and_expr  }
//	and_expr = not_expr  { AND not_expr  }
//	not_expr = NOT not_expr | primary
//	primary  = "(" expr ")"
//	         | EXISTS  ident
//	         | ident   IN     "(" string { "," string } ")"
//	         | ident   LIKE   string
//	         | ident   "="    string
//	         | ident   "!="   string
func Parse(expr string) (Predicate, error) {
	tokens, err := lex(expr)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	pred, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if !p.done() {
		return nil, fmt.Errorf("unexpected token %s at position %d", p.peek(), p.current().pos)
	}
	return pred, nil
}

// -------------------------------------------------------------------------
// Recursive descent parser
// -------------------------------------------------------------------------

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) peek() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return token{kind: tokEOF}
}

func (p *parser) current() token { return p.peek() }

func (p *parser) consume() token {
	t := p.peek()
	p.pos++
	return t
}

func (p *parser) done() bool { return p.peek().kind == tokEOF }

func (p *parser) expect(kind tokenKind, desc string) (token, error) {
	t := p.peek()
	if t.kind != kind {
		return token{}, fmt.Errorf("expected %s, got %s at position %d", desc, t, t.pos)
	}
	return p.consume(), nil
}

// or_expr = and_expr { OR and_expr }
func (p *parser) parseOr() (Predicate, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.consume()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &orPred{left, right}
	}
	return left, nil
}

// and_expr = not_expr { AND not_expr }
func (p *parser) parseAnd() (Predicate, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokAnd {
		p.consume()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &andPred{left, right}
	}
	return left, nil
}

// not_expr = NOT not_expr | primary
func (p *parser) parseNot() (Predicate, error) {
	if p.peek().kind == tokNot {
		p.consume()
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &notPred{inner}, nil
	}
	return p.parsePrimary()
}

// primary = "(" expr ")"
//
//	| EXISTS ident
//	| ident IN    "(" string { "," string } ")"
//	| ident LIKE  string
//	| ident "="   string
//	| ident "!="  string
func (p *parser) parsePrimary() (Predicate, error) {
	switch p.peek().kind {

	case tokLParen:
		p.consume()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokRParen, "')'"); err != nil {
			return nil, err
		}
		return inner, nil

	case tokExists:
		p.consume()
		ident, err := p.expect(tokIdent, "attribute name")
		if err != nil {
			return nil, err
		}
		return &existsPred{key: ident.val}, nil

	case tokIdent:
		ident := p.consume()
		return p.parseComparison(ident.val)

	default:
		t := p.peek()
		return nil, fmt.Errorf("unexpected token %s at position %d", t, t.pos)
	}
}

func (p *parser) parseComparison(key string) (Predicate, error) {
	op := p.peek()
	switch op.kind {

	case tokEq:
		p.consume()
		s, err := p.expectString()
		if err != nil {
			return nil, err
		}
		return &eqPred{key: key, val: s}, nil

	case tokNe:
		p.consume()
		s, err := p.expectString()
		if err != nil {
			return nil, err
		}
		return &nePred{key: key, val: s}, nil

	case tokLike:
		p.consume()
		pattern, err := p.expectString()
		if err != nil {
			return nil, err
		}
		rx, err := globToRegexp(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid LIKE pattern %q: %w", pattern, err)
		}
		return &likePred{key: key, pattern: pattern, rx: rx}, nil

	case tokIn:
		p.consume()
		if _, err := p.expect(tokLParen, "'('"); err != nil {
			return nil, err
		}
		vals := make(map[string]struct{})
		var reprs []string
		for {
			s, err := p.expectString()
			if err != nil {
				return nil, err
			}
			vals[s] = struct{}{}
			reprs = append(reprs, fmt.Sprintf("%q", s))
			if p.peek().kind != tokComma {
				break
			}
			p.consume() // consume comma
		}
		if _, err := p.expect(tokRParen, "')'"); err != nil {
			return nil, err
		}
		return &inPred{key: key, vals: vals, repr: strings.Join(reprs, ", ")}, nil

	default:
		return nil, fmt.Errorf("expected operator (=, !=, LIKE, IN) after %q, got %s at position %d", key, op, op.pos)
	}
}

func (p *parser) expectString() (string, error) {
	t, err := p.expect(tokString, "string literal")
	if err != nil {
		return "", err
	}
	return t.val, nil
}
