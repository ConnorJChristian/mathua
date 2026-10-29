package diagnostic

import (
	"strings"
	"testing"

	"github.com/chuma-beep/mathua/internal/concepts"
	"github.com/chuma-beep/mathua/internal/generator"
)

// bookOrPlain serves, at random, a book-reference question or an ordinary one
// when nothing rules the book out, as a question bank with both would.
type bookOrPlain struct{ n int }

func (g *bookOrPlain) Generate(ctx generator.GeneratorContext) generator.Problem {
	g.n++
	if !ctx.NoBookRefs && g.n%2 == 0 {
		h := generator.BookHeader(generator.BookRef{File: "b.pdf", Module: "m", Page: 4, Label: "Book Ex 1, #2"}, true)
		return generator.Problem{Question: h + "\n\nEnter it.", Answer: "1"}
	}
	return generator.Problem{Question: "plain?", Answer: "1"}
}

// Placement is objective and quick: it never asks for a page of the book.
func TestPlacementNeverServesBookRefs(t *testing.T) {
	d, err := concepts.Build([]concepts.Concept{{ID: "a", Label: "A", Domain: "d", Subdomain: "s",
		MasteryThreshold: concepts.MasteryThreshold{Streak: 3, AvgTimeSeconds: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	reg := generator.NewRegistry()
	if err := reg.Register("a", &bookOrPlain{}); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(d, reg)
	for i := 0; i < 20; i++ {
		p, _, err := e.NextQuestion(e.Start())
		if err != nil {
			t.Fatal(err)
		}
		if p == nil || strings.Contains(p.Question, generator.BookMarker) {
			t.Fatalf("placement served %+v", p)
		}
	}
}
