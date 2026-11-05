package statistics

// The numeric statistics concepts: spread, quartiles, z-scores, the standard error and margin of
// error, and conditional probability.
//
// The domain's spread and inference concepts were `symbolic`, and the corpus records what that means
// in production: the grading runtime has no SymPy, so a symbolic answer always grades incorrect.
// These eight are `numeric` instead, which is the same produced-answer shift machine learning needed
// and which section 4.2 of docs/content-expansion.md measures.
//
// Every answer is exact by construction: the data is symmetric about its mean so the variance is a
// whole number, the quartiles use the median of each half, the z-score uses a whole number of
// standard deviations, and the sample size is a perfect square so the standard error is whole.
import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/chuma-beep/mathua/internal/generator"
)

func registerNumeric(reg *generator.Registry) {
	reg.Register("stat.spread.mad", &madGen{})
	reg.Register("stat.spread.iqr", &iqrGen{})
	reg.Register("stat.quartile.value", &quartileGen{})
	reg.Register("stat.zscore.value", &zscoreGen{})
	reg.Register("stat.sampling.se", &standardErrorGen{})
	reg.Register("stat.infer.margin", &marginGen{})
	reg.Register("stat.prob.conditional", &conditionalGen{})
}

func smallInt(lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	return lo + rand.Intn(hi-lo+1)
}

// madGen asks for the mean absolute deviation, which is the average distance from the mean.
type madGen struct{}

func (g *madGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	// only the four-value shape: its deviations are all d, so the average is exact. The five-value
	// shape would give 6d/5, which integer division would truncate into a wrong answer.
	m := smallInt(5, 30)
	d := smallInt(1, 6)
	data := []int{m - d, m - d, m + d, m + d}
	sum := 0
	for _, x := range data {
		if x > m {
			sum += x - m
		} else {
			sum += m - x
		}
	}
	v := sum / len(data)
	return generator.Problem{
		Question: fmt.Sprintf("Find the mean absolute deviation of the data set \\(\\{%d, %d, %d, %d\\}\\).",
			data[0], data[1], data[2], data[3]),
		Answer: strconv.Itoa(v),
		Explanation: fmt.Sprintf("The mean is %d. Each value's distance from it is taken without a sign, and the average of those distances is %d. Taking absolute values rather than squares is what keeps the answer in the data's own units.",
			m, v),
		Facts: map[string]string{"mad": strconv.Itoa(v), "mean": strconv.Itoa(m)},
	}
}

// sortedSample returns an even-length sorted sample whose halves have whole medians.
func sortedSample() []int {
	n := 8 + 4*rand.Intn(2)
	vals := make([]int, n)
	start := smallInt(1, 10)
	for i := range vals {
		vals[i] = start + 2*i + 2*rand.Intn(2)
	}
	sort.Ints(vals)
	return vals
}

// iqrGen asks for the interquartile range, which is the spread of the middle half.
type iqrGen struct{}

func (g *iqrGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	vals := sortedSample()
	n := len(vals)
	lower := vals[:n/2]
	upper := vals[n/2:]
	q1 := (lower[len(lower)/2-1] + lower[len(lower)/2]) / 2
	q3 := (upper[len(upper)/2-1] + upper[len(upper)/2]) / 2
	return generator.Problem{
		Question: fmt.Sprintf("Find the interquartile range of the sorted data set \\(\\{%s\\}\\).", joinInts(vals)),
		Answer:   strconv.Itoa(q3 - q1),
		Explanation: fmt.Sprintf("The lower quartile is the median of the lower half, %d, and the upper quartile the median of the upper half, %d. Their difference, %d, is the spread of the middle half of the data and ignores the extremes.",
			q1, q3, q3-q1),
		Facts: map[string]string{"iqr": strconv.Itoa(q3 - q1), "q1": strconv.Itoa(q1), "q3": strconv.Itoa(q3)},
	}
}

// quartileGen asks for one quartile rather than the range.
type quartileGen struct{}

func (g *quartileGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	vals := sortedSample()
	n := len(vals)
	lower := vals[:n/2]
	upper := vals[n/2:]
	q1 := (lower[len(lower)/2-1] + lower[len(lower)/2]) / 2
	q3 := (upper[len(upper)/2-1] + upper[len(upper)/2]) / 2
	askQ1 := rand.Intn(2) == 0
	want, which, half := q3, "upper", upper
	if askQ1 {
		want, which, half = q1, "lower", lower
	}
	return generator.Problem{
		Question: fmt.Sprintf("Find the %s quartile of the sorted data set \\(\\{%s\\}\\).", which, joinInts(vals)),
		Answer:   strconv.Itoa(want),
		Explanation: fmt.Sprintf("Split the sorted data at its median, and take the median of the %s half: that half is \\(\\{%s\\}\\), whose middle pair averages to %d. The quartile is a median of a half, not a quarter of the range.",
			which, joinInts(half), want),
		Facts: map[string]string{"quartile": strconv.Itoa(want), "which": which},
	}
}

// zscoreGen asks how many standard deviations a value lies from the mean.
type zscoreGen struct{}

func (g *zscoreGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	mean := smallInt(20, 80)
	sd := smallInt(2, 10)
	k := smallInt(-3, 3)
	if k == 0 {
		k = 1
	}
	x := mean + k*sd
	return generator.Problem{
		Question: fmt.Sprintf("A distribution has mean %d and standard deviation %d. Find the z-score of the value %d.", mean, sd, x),
		Answer:   strconv.Itoa(k),
		Explanation: fmt.Sprintf("A z-score is the signed distance from the mean measured in standard deviations: (%d - %d) / %d = %d. A positive value is above the mean and a negative one below, and the size says how unusual it is.",
			x, mean, sd, k),
		Facts: map[string]string{"z": strconv.Itoa(k), "mean": strconv.Itoa(mean), "sd": strconv.Itoa(sd), "x": strconv.Itoa(x), "dev": strconv.Itoa(x - mean)},
	}
}

// standardErrorGen uses a sample size that is a perfect square, so the error is whole.
type standardErrorGen struct{}

func (g *standardErrorGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	n := smallInt(2, 6)
	n *= n
	s := n * smallInt(1, 5+int(ctx.Difficulty*8))
	root := 1
	for root*root < n {
		root++
	}
	return generator.Problem{
		Question: fmt.Sprintf("A sample of size %d has a standard deviation of %d. Find the standard error of the mean.", n, s),
		Answer:   strconv.Itoa(s / root),
		Explanation: fmt.Sprintf("The standard error is the standard deviation divided by the square root of the sample size: %d / sqrt(%d) = %d / %d = %d. Estimating the mean from more data is more precise, which is what the division by the root expresses.",
			s, n, s, root, s/root),
		Facts: map[string]string{"standard_error": strconv.Itoa(s / root), "size": strconv.Itoa(n), "sd": strconv.Itoa(s), "root": strconv.Itoa(root)},
	}
}

// marginGen multiplies a standard error by a critical value.
type marginGen struct{}

func (g *marginGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	se := smallInt(2, 12+int(ctx.Difficulty*10))
	z := 2
	if rand.Intn(2) == 0 {
		z = 3
	}
	return generator.Problem{
		Question: fmt.Sprintf("A confidence interval uses a critical value of %d and a standard error of %d. Find the margin of error.", z, se),
		Answer:   strconv.Itoa(z * se),
		Explanation: fmt.Sprintf("The margin of error is the critical value times the standard error: %d * %d = %d. The interval is then the estimate plus or minus that margin, so the margin is half its width.",
			z, se, z*se),
		Facts: map[string]string{"margin": strconv.Itoa(z * se), "standard_error": strconv.Itoa(se)},
	}
}

// conditionalGen asks for a conditional probability, which is a ratio of two probabilities given as
// fractions with a common denominator.
type conditionalGen struct{}

func (g *conditionalGen) Generate(ctx generator.GeneratorContext) generator.Problem {
	d := smallInt(2, 6)
	both := smallInt(1, d-1)
	b := both + smallInt(1, d-both)
	// P(A|B) = P(A and B) / P(B) = both/d over b/d = both/b
	num, den := reduceStat(both, b)
	return generator.Problem{
		Question: fmt.Sprintf("For two events, \\(P(A \\text{ and } B) = \\frac{%d}{%d}\\) and \\(P(B) = \\frac{%d}{%d}\\). Find \\(P(A \\mid B)\\).", both, d, b, d),
		Answer:   fmt.Sprintf("%d/%d", num, den),
		Explanation: fmt.Sprintf("Conditioning on B restricts attention to the cases where B happened, so the probability is the chance both happen over the chance B does: (%d/%d) / (%d/%d) = %d/%d.",
			both, d, b, d, num, den),
		Facts: conditionalFacts(num, den, both, b, d),
	}
}

// conditionalFacts publishes the walkthrough's values; "reduced" is set only
// when both/b reduces, so the schema's reduction step applies only then.
func conditionalFacts(num, den, both, b, d int) map[string]string {
	f := map[string]string{"out_num": strconv.Itoa(num), "out_den": strconv.Itoa(den), "both": strconv.Itoa(both), "b": strconv.Itoa(b), "d": strconv.Itoa(d)}
	if num != both {
		f["reduced"] = strconv.Itoa(num) + "/" + strconv.Itoa(den)
	}
	return f
}

// joinInts renders a slice of integers for a question's data set.
func joinInts(v []int) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ", ")
}

// reduceStat reduces a probability fraction.
func reduceStat(a, b int) (int, int) {
	g := a
	h := b
	for h != 0 {
		g, h = h, g%h
	}
	if g < 0 {
		g = -g
	}
	if g == 0 {
		return a, b
	}
	return a / g, b / g
}
