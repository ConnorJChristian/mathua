package pytrack

import (
	"os"
	"testing"
)

// The suite runs against the committed modules only, so it passes the same on
// every machine whatever that machine's modules-local/ holds.
func TestMain(m *testing.M) {
	os.Setenv("MATHUA_PUBLIC_MODULES_ONLY", "1")
	os.Exit(m.Run())
}
