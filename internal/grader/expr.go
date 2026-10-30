package grader

import (
	"math"
	"strings"
	"unicode"
)

// evalExpression reads a bare arithmetic expression such as "3-4" or "2*(1+3)".
//
// A learner solving x + 4 = 3 in their head often types the working rather than
// the result. The numeric grader called that unreadable, so a correct answer was
// reported as a formatting failure — one of the live reports from 23 October
// 2026 read "answered 3-4, marked wrong".
//
// Only plain arithmetic is accepted: numbers, + - * / ^, parentheses and
// spaces, with the multiplication and minus glyphs MathLive can emit. A string
// that does not parse entirely is refused, so "6,8" and "x+2" are still not
// numbers and fall through to whatever the caller does next.
func evalExpression(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	p := &exprParser{src: []rune(s)}
	v, ok := p.expr()
	if !ok {
		return 0, false
	}
	p.space()
	if p.pos != len(p.src) {
		return 0, false
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

type exprParser struct {
	src []rune
	pos int
}

func (p *exprParser) space() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
}

func (p *exprParser) peek() rune {
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return 0
}

// expr := term (('+' | '-') term)*
func (p *exprParser) expr() (float64, bool) {
	v, ok := p.term()
	if !ok {
		return 0, false
	}
	for {
		p.space()
		switch p.peek() {
		case '+':
			p.pos++
			r, ok := p.term()
			if !ok {
				return 0, false
			}
			v += r
		case '-', '\u2212':
			p.pos++
			r, ok := p.term()
			if !ok {
				return 0, false
			}
			v -= r
		default:
			return v, true
		}
	}
}

// term := unary (('*' | '/') unary)*
func (p *exprParser) term() (float64, bool) {
	v, ok := p.unary()
	if !ok {
		return 0, false
	}
	for {
		p.space()
		switch p.peek() {
		case '*', '\u00d7', '\u00b7':
			p.pos++
			r, ok := p.unary()
			if !ok {
				return 0, false
			}
			v *= r
		case '/', '\u00f7':
			p.pos++
			r, ok := p.unary()
			if !ok || r == 0 {
				return 0, false
			}
			v /= r
		default:
			return v, true
		}
	}
}

// unary := ('+' | '-') power | power — one leading sign, so "--42" stays
// unreadable rather than quietly meaning +42
func (p *exprParser) unary() (float64, bool) {
	p.space()
	switch p.peek() {
	case '-', '\u2212':
		p.pos++
		v, ok := p.power()
		return -v, ok
	case '+':
		p.pos++
		return p.power()
	}
	return p.power()
}

// power := primary ('^' unary)?
func (p *exprParser) power() (float64, bool) {
	base, ok := p.primary()
	if !ok {
		return 0, false
	}
	p.space()
	if p.peek() != '^' {
		return base, true
	}
	p.pos++
	exp, ok := p.unary()
	if !ok {
		return 0, false
	}
	return math.Pow(base, exp), true
}

// primary := number | '(' expr ')'
func (p *exprParser) primary() (float64, bool) {
	p.space()
	if p.peek() == '(' {
		p.pos++
		v, ok := p.expr()
		if !ok {
			return 0, false
		}
		p.space()
		if p.peek() != ')' {
			return 0, false
		}
		p.pos++
		return v, true
	}

	start := p.pos
	seenDigit := false
	for p.pos < len(p.src) {
		r := p.src[p.pos]
		if r >= '0' && r <= '9' {
			seenDigit = true
			p.pos++
			continue
		}
		if r == '.' && seenDigit {
			p.pos++
			continue
		}
		break
	}
	if !seenDigit {
		return 0, false
	}
	value, _, ok := parseNumeric(string(p.src[start:p.pos]))
	if !ok {
		return 0, false
	}
	return value, true
}
