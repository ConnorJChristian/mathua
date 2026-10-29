package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
)

// Fork change: questions whose prompt is a page of the learner's own book.
//
// A question-bank item or a proof problem may carry a BookRef instead of the
// text of a textbook exercise: the bank stores only the reference, the file
// and page, the answer key and an explanation in the bank's own words. The
// served question opens with BookMarker and a JSON description of the page,
// then a visible line naming the exercise and its page, so a host that does
// not know the marker still tells the learner where to read. Hosts that know
// it (web/next-app/lib/bookRef.ts) show the page in the embedded PDF reader
// directly above the answer box, or the label and page alone when the PDF is
// not installed.

// BookMarker opens a book-reference question; the JSON and "-->" follow.
const BookMarker = "<!--book-ref "

// BookRef points a question at an exercise in a PDF of a module's files/.
type BookRef struct {
	// File is the PDF, relative to the module's files/ folder.
	File string `json:"file"`
	// Module is the module whose files/ folder holds File.
	Module string `json:"module"`
	// Page is the PDF page the exercise starts on (1-based), EndPage the
	// last one when it runs over a page break.
	Page    int `json:"page"`
	EndPage int `json:"end_page,omitempty"`
	// PrintedPage is the page number printed in the book, for a learner
	// working from a paper copy. Optional.
	PrintedPage int `json:"printed_page,omitempty"`
	// Label names the exercise: "Hoy Exercise 5.4, #3".
	Label string `json:"label"`
}

// Validate checks the reference is complete.
func (b *BookRef) Validate() error {
	if b == nil {
		return nil
	}
	if strings.TrimSpace(b.File) == "" || strings.TrimSpace(b.Module) == "" || strings.TrimSpace(b.Label) == "" {
		return fmt.Errorf("book_ref needs file, module and label")
	}
	if strings.ContainsAny(b.Label, "\n") || strings.Contains(b.Label, "-->") {
		return fmt.Errorf("book_ref label must be one line")
	}
	if b.Page < 1 {
		return fmt.Errorf("book_ref page must be 1 or more")
	}
	if b.EndPage != 0 && b.EndPage < b.Page {
		return fmt.Errorf("book_ref end_page is before page")
	}
	if b.PrintedPage < 0 {
		return fmt.Errorf("book_ref printed_page must not be negative")
	}
	return nil
}

// bookRefWire is the JSON after BookMarker: the reference plus where the
// server found the file. URL is empty when the PDF is not installed.
type bookRefWire struct {
	BookRef
	Available bool   `json:"available"`
	URL       string `json:"url,omitempty"`
}

// BookFileURL is the path the server serves a module file at
// (server.handleModuleFile).
func BookFileURL(module, file string) string {
	segs := strings.Split(file, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return "/api/modules/" + url.PathEscape(module) + "/files/" + strings.Join(segs, "/")
}

// BookHeader is the start of a served book-reference question: the marker
// with the page as JSON, then a visible line with the label and the page.
// available reports whether the module's files/ folder has the PDF.
func BookHeader(b BookRef, available bool) string {
	w := bookRefWire{BookRef: b, Available: available}
	if available {
		w.URL = BookFileURL(b.Module, b.File)
	}
	js, _ := json.Marshal(w)
	// "-->" cannot occur in the JSON: Validate refuses it in the label and
	// json.Marshal escapes '>' in every other string.
	return BookMarker + string(js) + "-->\n\n" + BookLine(b, available)
}

// BookLine is the visible line naming the exercise and where it is.
func BookLine(b BookRef, available bool) string {
	where := fmt.Sprintf("PDF page %d", b.Page)
	if b.PrintedPage > 0 {
		where = fmt.Sprintf("book page %d, PDF page %d", b.PrintedPage, b.Page)
	}
	line := fmt.Sprintf("**%s** (%s).", b.Label, where)
	if !available {
		line += " The PDF is not installed, so work from your copy of the book."
	}
	return line
}

// History is what one learner has done with the questions of one topic,
// for gating (fork change). Implementations load lazily: a topic with no
// gated or book-reference items never asks.
type History interface {
	// Outcome reports whether the learner has answered the served problem
	// with this answer key and whether any of those answers was right. A
	// multiple-choice answer key names the option, not the item, so for one
	// the served question text identifies the problem as well.
	Outcome(answer, question string) (attempted, correct bool)
}

// GatedItems is implemented by question-bank generators whose items can be
// held back or preferred per learner (internal/pytrack). The engine's card
// mixing uses it for items filed under another topic.
type GatedItems interface {
	// ItemServable reports whether the item may be served under ctx: its
	// unlocks_after item is answered correctly, it is not a book reference
	// when ctx.NoBookRefs is set, and it is objective when ctx.ObjectiveOnly
	// is set.
	ItemServable(itemID string, ctx GeneratorContext) bool
	// ItemBookPending reports a book-reference item the learner has not yet
	// answered correctly, and whether they have tried it.
	ItemBookPending(itemID string, ctx GeneratorContext) (pending, attempted bool)
}

// BookPender is a generator that knows whether its topic still has a
// book-reference item this learner has not answered correctly. Card and proof
// mixers ask their base before serving anything else.
type BookPender interface {
	BookPending(ctx GeneratorContext) bool
}

// HasBookPending asks g, when it can answer.
func HasBookPending(g Generator, ctx GeneratorContext) bool {
	bp, ok := g.(BookPender)
	return ok && bp.BookPending(ctx)
}

// PickBook chooses which pending book-reference item to serve, given for
// each one, in book order, whether the learner has tried it. The first one
// never tried comes first; once every pending one has been tried, one of
// them is served half the time, so a missed exercise comes back without
// crowding out the rest of the topic. -1 means serve something else.
func PickBook(attempted []bool) int {
	for i, a := range attempted {
		if !a {
			return i
		}
	}
	if len(attempted) > 0 && rand.Intn(2) == 0 {
		return rand.Intn(len(attempted))
	}
	return -1
}

// QuestionKey is the part of a served question that identifies it across
// serves: without the card marker the engine puts before a card served on
// another topic, and without a book header, whose availability note changes
// when the PDF is installed or removed.
func QuestionKey(q string) string {
	q = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(q), "<!--card-->"))
	if strings.HasPrefix(q, BookMarker) {
		if i := strings.Index(q, "-->"); i >= 0 {
			q = strings.TrimLeft(q[i+3:], "\n")
			if j := strings.Index(q, "\n\n"); j >= 0 {
				q = q[j+2:]
			} else {
				q = ""
			}
		}
	}
	return strings.TrimSpace(q)
}
