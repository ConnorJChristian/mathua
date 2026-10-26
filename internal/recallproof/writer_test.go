package recallproof

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/proof"
	"github.com/chuma-beep/mathua/internal/pytrack"
)

func fakeMessages(t *testing.T, reply func(body map[string]any) (int, string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "k" || r.Header.Get("anthropic-version") == "" {
			w.WriteHeader(400)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		json.Unmarshal(raw, &body)
		st, b := reply(body)
		w.WriteHeader(st)
		io.WriteString(w, b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func message(text, stop string) string {
	raw, _ := json.Marshal(map[string]any{
		"type": "message", "stop_reason": stop,
		"content": []map[string]any{{"type": "text", "text": text}},
		"usage":   map[string]any{"input_tokens": 900, "output_tokens": 700},
	})
	return string(raw)
}

func TestAnthropicWriter(t *testing.T) {
	var sawCard bool
	scheme := `{"gradable": true, "note": "", "reference": "Write the odd numbers as 2a+1 and 2b+1.", "given": "", "milestones": [{"id": "m1", "points": 3, "desc": "writes both as 2k+1"}, {"id": "m2", "points": 4, "desc": "factors out 2"}], "caps": [], "alternatives": []}`
	srv := fakeMessages(t, func(body map[string]any) (int, string) {
		msgs, _ := json.Marshal(body["messages"])
		sawCard = strings.Contains(string(msgs), "sum of two odd numbers") && strings.Contains(string(msgs), "DECK: anki_linear_algebra")
		return 200, message("Here it is:\n"+scheme, "end_turn")
	})
	w := &AnthropicWriter{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 3000}
	s, u, err := w.WriteScheme(context.Background(), card())
	if err != nil || len(s.Milestones) != 2 || u.OutputTokens != 700 || !sawCard {
		t.Fatalf("scheme: %v %+v %+v card sent: %v", err, s, u, sawCard)
	}

	cut := fakeMessages(t, func(map[string]any) (int, string) {
		return 200, message(`{"gradable": true, "reference": "half`, "max_tokens")
	})
	w.BaseURL = cut.URL
	if _, u, err := w.WriteScheme(context.Background(), card()); !errors.Is(err, ErrUnparseable) || u.OutputTokens != 700 {
		t.Errorf("a cut-off reply is unparseable and still billed: %v %+v", err, u)
	}
	down := fakeMessages(t, func(map[string]any) (int, string) {
		return 529, `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`
	})
	w.BaseURL = down.URL
	if _, _, err := w.WriteScheme(context.Background(), card()); err == nil || !strings.Contains(err.Error(), "overloaded") {
		t.Errorf("HTTP error: %v", err)
	}
	if _, _, err := (&AnthropicWriter{}).WriteScheme(context.Background(), card()); err == nil {
		t.Error("no key must fail")
	}
}

// variantWriter is a fake writer that also writes wrong answers.
type variantWriter struct{ fakeWriter }

func (v *variantWriter) WrongAnswer(context.Context, pytrack.RecallCard, *Scheme) (string, proof.Usage, error) {
	return "Odd plus odd is odd, so the sum is odd.", proof.Usage{InputTokens: 500, OutputTokens: 200}, nil
}

func TestCalibrateRecall(t *testing.T) {
	store := proof.NewMemStore()
	p, _ := newProof(t, store, "k")
	vw := &variantWriter{}
	g := New(p, on(), vw)
	a, b := card(), card()
	b.Key, b.Back = "anki.x.a#v", "CORRECT: (2a+1)+(2b+1) = 2(a+b+1)."
	var out strings.Builder
	res, m := g.Calibrate(context.Background(), []pytrack.RecallCard{a, b}, g.VariantWriter(), &out)
	if len(res) != 2 || m.Cards != 2 {
		t.Fatalf("%+v", m)
	}
	// The fake judge passes only answers containing CORRECT: card a's back
	// fails (a false fail), card b's passes, and both wrong answers fail.
	if m.BackPass != 1 || m.BackFail != 1 || m.WrongPass != 0 || m.WrongFail != 2 || m.Ungraded != 0 {
		t.Errorf("matrix: %+v", m)
	}
	want := 2*p.Cfg.Prices.Cost(writerUsage) + 2*p.Cfg.Prices.Cost(proof.Usage{InputTokens: 500, OutputTokens: 200}) + 4*3*p.Cfg.Prices.Cost(judgeUsage)
	if m.USD < want-1e-9 || m.USD > want+1e-9 {
		t.Errorf("cost %.6f, want %.6f (schemes, wrong answers and gradings)", m.USD, want)
	}
	if !strings.Contains(m.String(), "false-pass 0%") || !strings.Contains(out.String(), "anki.x.a#v") {
		t.Errorf("report:\n%s\n%s", m, out.String())
	}
}
