package server

import (
	"testing"

	"github.com/chuma-beep/mathua/internal/pytrack"
)

func TestCardMix(t *testing.T) {
	items := func(kinds ...string) []pytrack.Item {
		out := make([]pytrack.Item, len(kinds))
		for i, k := range kinds {
			out[i] = pytrack.Item{Kind: k}
		}
		return out
	}
	cases := []struct {
		kinds []string
		want  string
	}{
		{[]string{"recall", "recall"}, "2 cards. For a flip card, answer in your head, reveal the back, then say whether you had it."},
		{[]string{"math"}, "1 card. Type an answer and Mathua checks it; any equivalent form counts."},
		{[]string{"math", "math", "mc", "recall"}, "4 cards: 2 you type, 1 you choose, 1 you flip. Type an answer and Mathua checks it; any equivalent form counts. For a flip card, answer in your head, reveal the back, then say whether you had it."},
	}
	for _, c := range cases {
		if got := cardMix(items(c.kinds...)); got != c.want {
			t.Errorf("%v:\n got %q\nwant %q", c.kinds, got, c.want)
		}
	}
}
