package recallproof

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the written-answer flashcard grader's own configuration. The
// judge, its model, runs, prices and the monthly budget are the proof
// grader's (MATHUA_PROOF_*).
type Config struct {
	// On is MATHUA_LLM_RECALL: on by default whenever an API key is set,
	// "0" (or false/off/no) turns it off.
	On bool
	// Banks is MATHUA_LLM_RECALL_BANKS, an optional allowlist of bank files;
	// Exclude is MATHUA_LLM_RECALL_EXCLUDE_BANKS. Both are comma-separated
	// bank file names with or without ".json" and the "anki_" prefix, so
	// "linear_algebra" names anki_linear_algebra.json.
	Banks   []string
	Exclude []string
	// SchemeMaxTokens bounds one marking scheme's reply
	// (MATHUA_LLM_RECALL_SCHEME_MAX_TOKENS).
	SchemeMaxTokens int
	// SchemeTimeout bounds writing one marking scheme.
	SchemeTimeout time.Duration
	// RetryAfter is how long a card whose scheme failed for a passing reason
	// (a network error, an unreadable reply) stays self-graded before the
	// next attempt (MATHUA_LLM_RECALL_RETRY_HOURS). A card the model judged
	// ungradable as a written answer stays self-graded until it is edited.
	RetryAfter time.Duration
}

// Defaults.
const (
	DefaultSchemeMaxTokens = 3000
	DefaultSchemeTimeout   = 120 * time.Second
	DefaultRetryAfter      = 24 * time.Hour
)

// ConfigFromEnv reads MATHUA_LLM_RECALL*. hasKey says whether
// ANTHROPIC_API_KEY is set: without it the grader is off whatever
// MATHUA_LLM_RECALL says.
func ConfigFromEnv(hasKey bool) Config {
	c := Config{
		On:              hasKey,
		Banks:           bankList(os.Getenv("MATHUA_LLM_RECALL_BANKS")),
		Exclude:         bankList(os.Getenv("MATHUA_LLM_RECALL_EXCLUDE_BANKS")),
		SchemeMaxTokens: DefaultSchemeMaxTokens,
		SchemeTimeout:   DefaultSchemeTimeout,
		RetryAfter:      DefaultRetryAfter,
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MATHUA_LLM_RECALL"))) {
	case "0", "false", "off", "no":
		c.On = false
	}
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("MATHUA_LLM_RECALL_SCHEME_MAX_TOKENS"))); err == nil && n >= 500 {
		c.SchemeMaxTokens = n
	}
	if h, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv("MATHUA_LLM_RECALL_RETRY_HOURS")), 64); err == nil && h >= 0 {
		c.RetryAfter = time.Duration(h * float64(time.Hour))
	}
	return c
}

func (c Config) normalized() Config {
	if c.SchemeMaxTokens < 500 {
		c.SchemeMaxTokens = DefaultSchemeMaxTokens
	}
	if c.SchemeTimeout <= 0 {
		c.SchemeTimeout = DefaultSchemeTimeout
	}
	return c
}

// bankList splits a comma-separated list of bank names.
func bankList(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if b := bankName(f); b != "" {
			out = append(out, b)
		}
	}
	return out
}

// bankName is a bank's name for matching: lower case, without ".json" or the
// "anki_" prefix.
func bankName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".json")
	return strings.TrimPrefix(s, "anki_")
}

// BankAllowed reports whether cards in this bank file may be graded:
// allowlisted (or no allowlist) and not excluded.
func (c Config) BankAllowed(bank string) bool {
	b := bankName(bank)
	for _, x := range c.Exclude {
		if x == b {
			return false
		}
	}
	if len(c.Banks) == 0 {
		return true
	}
	for _, x := range c.Banks {
		if x == b {
			return true
		}
	}
	return false
}

// SameBank reports whether two bank names name the same bank file
// ("linear_algebra" and "anki_linear_algebra.json" do).
func SameBank(a, b string) bool { return bankName(a) == bankName(b) }
