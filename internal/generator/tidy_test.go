package generator

import "testing"

func TestTidy(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Solve: $|x + -3| = 5$", "Solve: $|x - 3| = 5$"},
		{"$f(x) = 1x^{2} + 1$. Find $f(2)$.", "$f(x) = x^{2} + 1$. Find $f(2)$."},
		{"Find the roots of $x^{2} + -1x + 0 = 0$.", "Find the roots of $x^{2} - x = 0$."},
		{"divide $(1x^{2} + 2x)$ by $(x - -1)$", "divide $(x^{2} + 2x)$ by $(x + 1)$"},
		{"Is $y = -1x - 2$ a function?", "Is $y = -x - 2$ a function?"},
		{"The monic polynomial x^3 + -1x^2 + -10x + -8 has", "The monic polynomial x^3 - x^2 - 10x - 8 has"},
		{"Find $\\int_{0}^{10} 1 x^{3} \\, dx$.", "Find $\\int_{0}^{10} x^{3} \\, dx$."},
		{"For $1y'' +0y' +6y = 0$, what", "For $y'' +6y = 0$, what"},
		{"has the complex root 5-1i. A) 5+1i", "has the complex root 5-i. A) 5+i"},
		{"Let f(x) = (1x + 2)^4. Using", "Let f(x) = (x + 2)^4. Using"},
		{"$\\frac{4n^2 + 3n + 7}{2n^2 + 1n + 0}$", "$\\frac{4n^2 + 3n + 7}{2n^2 + n}$"},
		{"$f(x)=1x^{2}y+1xy^{2}$", "$f(x)=x^{2}y+xy^{2}$"},
		{"Parabola: $(y+0)^{2} = -8(x-1)$.", "Parabola: $y^{2} = -8(x-1)$."},
		{"$f(x)^{2} + g(x)$", "$f(x)^{2} + g(x)$"},
		{"Is $(1,1)$ a solution to $y < 1x + 1$?", "Is $(1,1)$ a solution to $y < x + 1$?"},
		{"Integrating 2x in x gives 1x^2, and", "Integrating 2x in x gives x^2, and"},
		{"the product 1y(4 - y) is zero", "the product y(4 - y) is zero"},
		{"vanishes: 1x - 4 = 0 gives x = 4.", "vanishes: x - 4 = 0 gives x = 4."},
		// Untouched: no variable, words, numbers, decimals, units.
		{"A neuron computes $1 \\cdot 2 + -1 \\cdot 1 + -2$ and", "A neuron computes $1 \\cdot 2 + -1 \\cdot 1 + -2$ and"},
		{"holding 1s and 0s", "holding 1s and 0s"},
		{"A rectangle's length is 1 m more than 3 times", "A rectangle's length is 1 m more than 3 times"},
		{"$x + 0.5 = 1e-5$", "$x + 0.5 = 1e-5$"},
		{"$11x + 21y = 10$ and $x^{1}$", "$11x + 21y = 10$ and $x^{1}$"},
		{"It costs \\$1 and $x + -2$", "It costs \\$1 and $x - 2$"},
		{"$$\nx^{2} + -4\n$$", "$$\nx^{2} - 4\n$$"},
	}
	for _, c := range cases {
		if got := Tidy(c.in); got != c.want {
			t.Errorf("Tidy(%q)\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestTidyExplanation(t *testing.T) {
	if got := TidyExplanation("gives $(1 - -7/25)/2 = 16/25$."); got != "gives $(1 + 7/25)/2 = 16/25$." {
		t.Errorf("got %q", got)
	}
	if got := Tidy("Compute $(1 - -7/25)/2$."); got != "Compute $(1 - -7/25)/2$." {
		t.Errorf("a question keeps its arithmetic: %q", got)
	}
}

func TestTidyAnswer(t *testing.T) {
	cases := map[string]string{
		"4x + -2":             "4x - 2",
		"y=1x+-1":             "y=x-1",
		"x=-2+3t,y=4+-1t":     "x=-2+3t,y=4-t",
		"1x^2+C":              "x^2+C",
		"25-1i":               "25-i",
		"(4/1)e^(1x)+C":       "(4/1)e^(x)+C",
		"C1*e^(3x)+C2*e^(1x)": "C1*e^(3x)+C2*e^(x)",
		"y = 1x - 3":          "y = x - 3",
		"12":                  "12",
		"1/11":                "1/11",
	}
	for in, want := range cases {
		if got := TidyAnswer(in); got != want {
			t.Errorf("TidyAnswer(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTidyKeepsSignedOperandDrills(t *testing.T) {
	for _, id := range []string{"arith.neg.add", "prealg.integers.subtract"} {
		if !keepsSignedOperands(id) {
			t.Errorf("%s should keep 4 - -3 as written", id)
		}
	}
}
