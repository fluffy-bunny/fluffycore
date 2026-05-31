// Package cefilter compiles a human-readable filter expression into an
// efficient predicate tree that evaluates against a flat key/value attribute
// map (e.g. CloudEvent envelope attributes + extensions).
//
// Syntax:
//
//	type = "com.example.order"
//	source LIKE "checkout/*"
//	orgid IN ("org-1", "org-2")
//	EXISTS priority
//	(type = "com.example.*" OR type = "com.acme.*") AND orgid = "org-1" AND NOT EXISTS debug
package cefilter

import (
	"fmt"
	"regexp"
	"strings"
)

// Attrs is the flat view of a CloudEvent: standard attributes + extensions,
// all coerced to strings. Build one with AttrMap or your own accessor.
type Attrs map[string]string

// Predicate is the compiled, allocation-free evaluator.
// Once built via Parse, call Match on every incoming event.
type Predicate interface {
	Match(a Attrs) bool
	String() string // human-readable reconstruction (for debugging)
}

// -------------------------------------------------------------------------
// Leaf predicates
// -------------------------------------------------------------------------

type eqPred struct{ key, val string }

func (p *eqPred) Match(a Attrs) bool { v, ok := a[p.key]; return ok && v == p.val }
func (p *eqPred) String() string     { return fmt.Sprintf("%s = %q", p.key, p.val) }

type nePred struct{ key, val string }

func (p *nePred) Match(a Attrs) bool { v, ok := a[p.key]; return !ok || v != p.val }
func (p *nePred) String() string     { return fmt.Sprintf("%s != %q", p.key, p.val) }

type likePred struct {
	key     string
	pattern string        // original glob, kept for String()
	rx      *regexp.Regexp // compiled once at parse time
}

func (p *likePred) Match(a Attrs) bool { v, ok := a[p.key]; return ok && p.rx.MatchString(v) }
func (p *likePred) String() string     { return fmt.Sprintf("%s LIKE %q", p.key, p.pattern) }

type inPred struct {
	key  string
	vals map[string]struct{} // O(1) lookup
	repr string              // for String()
}

func (p *inPred) Match(a Attrs) bool {
	v, ok := a[p.key]
	if !ok {
		return false
	}
	_, found := p.vals[v]
	return found
}
func (p *inPred) String() string { return fmt.Sprintf("%s IN (%s)", p.key, p.repr) }

type existsPred struct{ key string }

func (p *existsPred) Match(a Attrs) bool { _, ok := a[p.key]; return ok }
func (p *existsPred) String() string     { return fmt.Sprintf("EXISTS %s", p.key) }

// -------------------------------------------------------------------------
// Composite predicates — short-circuit evaluated
// -------------------------------------------------------------------------

type andPred struct{ left, right Predicate }

func (p *andPred) Match(a Attrs) bool { return p.left.Match(a) && p.right.Match(a) }
func (p *andPred) String() string     { return fmt.Sprintf("(%s AND %s)", p.left, p.right) }

type orPred struct{ left, right Predicate }

func (p *orPred) Match(a Attrs) bool { return p.left.Match(a) || p.right.Match(a) }
func (p *orPred) String() string     { return fmt.Sprintf("(%s OR %s)", p.left, p.right) }

type notPred struct{ inner Predicate }

func (p *notPred) Match(a Attrs) bool { return !p.inner.Match(a) }
func (p *notPred) String() string     { return fmt.Sprintf("NOT %s", p.inner) }

// -------------------------------------------------------------------------
// Glob → regexp compiler (runs once at parse time, never per event)
// -------------------------------------------------------------------------

// globToRegexp converts a simple glob pattern (*, ?) to a compiled regexp.
// '*' matches any sequence of characters; '?' matches exactly one character.
func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var sb strings.Builder
	sb.WriteString(`\A`) // anchor start
	for _, ch := range pattern {
		switch ch {
		case '*':
			sb.WriteString(`.*`)
		case '?':
			sb.WriteString(`.`)
		default:
			sb.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	sb.WriteString(`\z`) // anchor end
	return regexp.Compile(sb.String())
}
