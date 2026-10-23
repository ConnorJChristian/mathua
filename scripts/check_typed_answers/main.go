// Command check_typed_answers grades imported typed answers with Mathua's own
// graders before they become questions. scripts/import_anki.py runs it.
//
//	go run ./scripts/check_typed_answers < candidates.jsonl > verdicts.jsonl
//
// Each input line is {"id", "grading", "expected", "ref", "wrong"}:
//
//   - expected is what Mathua's math editor sends for the book's answer,
//   - ref is an independent reading of the same LaTeX (SymPy's parser),
//   - wrong is ref plus one.
//
// A candidate passes when ref is graded correct against expected and wrong is
// graded wrong. Each output line is {"id", "ok", "reason"}. Run it from the
// repository root, with a python3 that has SymPy on PATH, so the expression
// grader finds grading/sympy_service.py.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/chuma-beep/mathua/internal/grader"
)

type candidate struct {
	ID       string `json:"id"`
	Grading  string `json:"grading"`
	Expected string `json:"expected"`
	Ref      string `json:"ref"`
	Wrong    string `json:"wrong"`
}

type verdict struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

func check(r *grader.Router, c candidate) verdict {
	v := verdict{ID: c.ID}
	if c.Expected == "" {
		v.Reason = "the math editor produced nothing for this LaTeX"
		return v
	}
	gt := grader.GradingType(c.Grading)
	right := r.Grade(gt, c.Expected, c.Ref)
	if right.Unavailable {
		v.Reason = "grader unavailable: " + right.Feedback
		return v
	}
	if !right.Correct {
		v.Reason = "the book's answer is not graded correct: " + right.Feedback
		return v
	}
	wrong := r.Grade(gt, c.Expected, c.Wrong)
	if wrong.Correct {
		v.Reason = "a wrong answer is graded correct"
		return v
	}
	v.OK = true
	return v
}

func main() {
	r := grader.NewRouter()
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<20)
	var cands []candidate
	for in.Scan() {
		var c candidate
		if err := json.Unmarshal(in.Bytes(), &c); err != nil {
			fmt.Fprintln(os.Stderr, "bad line:", err)
			os.Exit(2)
		}
		cands = append(cands, c)
	}
	out := make([]verdict, len(cands))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i := range cands {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			out[i] = check(r, cands[i])
		}(i)
	}
	wg.Wait()
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	enc := json.NewEncoder(w)
	for _, v := range out {
		_ = enc.Encode(v)
	}
}
