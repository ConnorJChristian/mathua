package proof

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Defaults. The model and the number of runs are the two levers on cost and
// on agreement with a human grader; both are environment settings so they can
// be changed after calibration without a rebuild.
const (
	DefaultModel       = "claude-sonnet-5-5"
	DefaultRuns        = 5
	DefaultMonthlyUSD  = 60.0
	DefaultMaxTokens   = 2000
	DefaultTimeout     = 120 * time.Second
	DefaultAPIBase     = "https://api.anthropic.com"
	DefaultProofShare  = 0.2
	AnthropicVersion   = "2023-06-01"
	MaxScore           = 7
	PassMark           = 6
	maxRuns            = 9
	maxProofCharacters = 20000
)

// Prices are US dollars per million tokens. A cache write is billed at
// CacheWriteMult times the input price and a cache read at CacheReadMult.
type Prices struct {
	InputPerMTok   float64
	OutputPerMTok  float64
	CacheWriteMult float64
	CacheReadMult  float64
}

// defaultPrices deliberately err high: the spend guard is a ceiling, and a
// price that is too high stops grading a little early while one that is too
// low lets the month overrun. Set MATHUA_PROOF_PRICE_IN / _OUT to the exact
// list prices of the model in use.
func defaultPrices(model string) Prices {
	p := Prices{InputPerMTok: 3, OutputPerMTok: 15, CacheWriteMult: 1.25, CacheReadMult: 0.1}
	if strings.Contains(model, "opus") {
		p.InputPerMTok, p.OutputPerMTok = 5, 25
	}
	return p
}

// Config is the proof grader's configuration.
type Config struct {
	APIKey     string
	APIBase    string
	Model      string
	Runs       int
	MaxTokens  int
	Timeout    time.Duration
	MonthlyUSD float64
	Prices     Prices
	// CountsForMastery lets a fully correct proof count as mastery evidence.
	//
	// It is off by default, and should stay off until the judge has been
	// calibrated against your own grading (mathua proof calibrate). The judge
	// is a sampled language model: its error rate on your proofs is unknown
	// until measured, and the costly error is leniency, a flawed proof passed
	// as correct. Mastery evidence is what moves a topic up the ladder and
	// spaces its reviews out, so an uncalibrated false pass would retire a
	// topic you have not learned, and nothing downstream would notice. While
	// off, a proof result is feedback only: it is recorded as an attempt for
	// your history but is treated like a self-graded card (never evidence,
	// never a miss toward a halt).
	CountsForMastery bool
	// Share is the fraction of a topic's practice questions that are proof
	// problems, when the topic has any (MATHUA_PROOF_SHARE, 0 to 1).
	Share float64
}

// ConfigFromEnv reads the MATHUA_PROOF_* settings and ANTHROPIC_API_KEY.
func ConfigFromEnv() Config {
	c := Config{
		APIKey:     strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		APIBase:    envString("MATHUA_PROOF_API_BASE", DefaultAPIBase),
		Model:      envString("MATHUA_PROOF_MODEL", DefaultModel),
		Runs:       envInt("MATHUA_PROOF_RUNS", DefaultRuns),
		MaxTokens:  envInt("MATHUA_PROOF_MAX_TOKENS", DefaultMaxTokens),
		Timeout:    time.Duration(envInt("MATHUA_PROOF_TIMEOUT_SECONDS", int(DefaultTimeout/time.Second))) * time.Second,
		MonthlyUSD: envFloat("MATHUA_PROOF_MONTHLY_USD", DefaultMonthlyUSD),
		Share:      envFloat("MATHUA_PROOF_SHARE", DefaultProofShare),
	}
	c.Prices = defaultPrices(c.Model)
	c.Prices.InputPerMTok = envFloat("MATHUA_PROOF_PRICE_IN", c.Prices.InputPerMTok)
	c.Prices.OutputPerMTok = envFloat("MATHUA_PROOF_PRICE_OUT", c.Prices.OutputPerMTok)
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MATHUA_PROOF_COUNTS_FOR_MASTERY"))) {
	case "1", "true", "yes", "on":
		c.CountsForMastery = true
	}
	return c.normalized()
}

func (c Config) normalized() Config {
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.APIBase == "" {
		c.APIBase = DefaultAPIBase
	}
	c.APIBase = strings.TrimRight(c.APIBase, "/")
	if c.Runs < 1 {
		c.Runs = 1
	}
	if c.Runs > maxRuns {
		c.Runs = maxRuns
	}
	if c.MaxTokens < 256 {
		c.MaxTokens = DefaultMaxTokens
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Share < 0 || c.Share > 1 {
		c.Share = DefaultProofShare
	}
	if c.Prices == (Prices{}) {
		c.Prices = defaultPrices(c.Model)
	}
	return c
}

// Cost is the dollar cost of one response's usage at these prices.
func (p Prices) Cost(u Usage) float64 {
	in := float64(u.InputTokens) + float64(u.CacheCreationInputTokens)*p.CacheWriteMult + float64(u.CacheReadInputTokens)*p.CacheReadMult
	return (in*p.InputPerMTok + float64(u.OutputTokens)*p.OutputPerMTok) / 1e6
}

func envString(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(k))); err == nil {
		return v
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(k)), 64); err == nil && v >= 0 {
		return v
	}
	return def
}
