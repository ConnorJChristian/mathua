package grader

import (
	"math/rand"
	"strings"
	"testing"
)

// NormalizeTypedMath takes whatever a learner typed into the LaTeX box when
// the converter has not loaded: it must never panic, and the common forms
// must come out as the plain spellings the graders read (fork check).
func TestNormalizeTypedMathRoundTrips(t *testing.T) {
	r := NewRouter()
	cases := []struct{ latex, plain string }{
		{`\frac{1}{2}`, "1/2"},
		{`-\frac{3}{4}`, "-3/4"},
		{`\dfrac{x^{2}}{3}`, "x^2/3"},
		{`3x^{2}+2x`, "3x^2+2x"},
		{`\sqrt{x^{2}+5}`, "sqrt(x^2+5)"},
		{`2\cdot x`, "2*x"},
		{`\pi x`, "pi*x"},
		{`e^{x}`, "e^x"},
		{`\left(x+1\right)^{2}`, "(x+1)^2"},
		{`x_{1}+x_{2}`, "x_1+x_2"},
		{`\beta_{1}`, "beta_1"},
		{`−2`, "-2"},
		{`3x²`, "3x^2"},
		{`\frac{1}{x}\,dx`, "1/x dx"},
		{`0.5\times 10`, "0.5*10"},
		{`x^{-1}`, "x^(-1)"},
		{`e^{-2}`, "exp(-2)"},
		{`1-e^{-20}`, "1-exp(-20)"},
		{`e^-2`, "e^(-2)"},
		{`\frac{\sqrt{2}}{2}`, "sqrt(2)/2"},
	}
	for _, c := range cases {
		got := NormalizeTypedMath(c.latex, c.plain)
		if !r.Grade(GradingExpression, c.plain, got).Correct {
			t.Errorf("%s -> %q does not grade as %q", c.latex, got, c.plain)
		}
	}
}

// A few hundred generated LaTeX strings, deep braces and all: no panic, no
// hang, and a string that was plain already comes back unchanged apart from
// spacing.
func TestNormalizeTypedMathNeverPanics(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	atoms := []string{"x", "y", "2", "10", "0.5", `\pi`, `\beta`, "β", "e", `\alpha_{1}`, "x_{2}", "−1", "²"}
	wellFormed := []func(a, b string) string{
		func(a, b string) string { return `\frac{` + a + `}{` + b + `}` },
		func(a, b string) string { return a + "^{" + b + "}" },
		func(a, b string) string { return `\sqrt{` + a + `}` + b },
		func(a, b string) string { return a + `\cdot ` + b },
		func(a, b string) string { return `\left(` + a + `+` + b + `\right)` },
		func(a, b string) string { return a + `\,` + b },
		func(a, b string) string { return a + "{:" + b + ":}" },
		func(a, b string) string { return a + "_{" + b + "}" },
	}
	malformed := append([]func(a, b string) string{
		func(a, b string) string { return "{" + a + "}{" + b },
		func(a, b string) string { return a + "}" + b + "^{" },
		func(a, b string) string { return `\frac{` + a + "}" },
		func(a, b string) string { return a + "^{}" + b },
	}, wellFormed...)
	var gen func(wrap []func(a, b string) string, depth int) string
	gen = func(wrap []func(a, b string) string, depth int) string {
		if depth == 0 || rng.Intn(3) == 0 {
			return atoms[rng.Intn(len(atoms))]
		}
		return wrap[rng.Intn(len(wrap))](gen(wrap, depth-1), gen(wrap, depth-1))
	}
	for i := 0; i < 600; i++ {
		wrap := malformed
		if i%2 == 0 {
			wrap = wellFormed
		}
		s := gen(wrap, 1+rng.Intn(5))
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Fatalf("panic on %q: %v", s, e)
				}
			}()
			out := NormalizeTypedMath(s, s)
			if i%2 == 0 {
				// Well-formed input: every LaTeX form is read. Nesting deeper
				// than the rewrite's passes is the one excuse.
				for _, tex := range []string{`\frac`, `\sqrt`, `^{`, `_{`, `\cdot`, `\pi`, `\beta`, `\left`, "{:", "²", "−"} {
					if strings.Contains(out, tex) && strings.Count(s, "{") < 16 {
						t.Errorf("%q left %s in %q", s, tex, out)
					}
				}
			}
			_ = NewRouter().Grade(GradingExpression, out, out)
		}()
	}
	for _, plain := range []string{"3x^2+2x", "1/2", "-7", "x_1*y", "sqrt(2)/2", "2*pi*r"} {
		if got := NormalizeTypedMath(plain, plain); strings.Join(strings.Fields(got), "") != strings.Join(strings.Fields(plain), "") {
			t.Errorf("plain %q changed to %q", plain, got)
		}
	}
}

// Forms the math editor exports and the LaTeX box sends that used to be
// refused (fork fix, from grading every key's editor form).
func TestNormalizeTypedMathEditorForms(t *testing.T) {
	r := NewRouter()
	for _, c := range []struct{ typed, plain string }{
		{"100-4∗q", "100-4*q"},
		{"(20)/(3root(3)(L))", "20/(3*L^(1/3))"},
		{`\frac{20}{3\sqrt[3]{L}}`, "20/(3*L^(1/3))"},
		{"(10root(4)(24))/(3)", "10*24^(1/4)/3"},
		{"-(f_(L K))/(p(f_(K K)f_(L L)-f_(L K)^2))", "-f_LK/(p*(f_KK*f_LL - f_LK^2))"},
		{`\frac34`, "3/4"},
		{`-\frac14`, "-1/4"},
		{`\frac1{12}`, "1/12"},
		{`\frac{2\ln x}{x}`, "2*ln(x)/x"},
		{`\ln|x+8|`, "ln|x+8|"},
		{`x^{3}\cos x+3x^{2}\sin x`, "x^3*cos(x)+3x^2*sin(x)"},
		{`b_{1} \log{\left(x_{1} \right)} + \log{\left(a \right)}`, "b_1*log(x_1) + log(a)"},
		{`1 + \frac{\log{\left(2 \right)}}{\log{\left(b \right)}}`, "1 + log(2)/log(b)"},
	} {
		got := NormalizeTypedMath(c.typed, c.plain)
		if !r.Grade(GradingExpression, c.plain, got).Correct {
			t.Errorf("%s -> %q does not grade as %q", c.typed, got, c.plain)
		}
	}
}
