package proof

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Calibration measures the judge against human scores before its results are
// trusted (see Config.CountsForMastery). A calibration file is a CSV with a
// header row and the columns problem_id, proof_text and human_score (0-7); an
// optional fourth column, label, is carried into the report.

// CalRow is one proof with the score a human gave it.
type CalRow struct {
	ProblemID string
	Proof     string
	Human     int
	Label     string
}

// CalResult is the judge's grade of one row.
type CalResult struct {
	Row   CalRow
	Judge int
	Pass  bool // fully correct by the judge
	Runs  []int
	// Err is why the row was not graded, or "".
	Err string
	USD float64
}

// CalStats summarises a calibration run over the graded rows.
type CalStats struct {
	N        int // graded rows
	Ungraded int
	// MAE is the mean absolute difference judge - human, in points of 7.
	MAE float64
	// Within1 is the share of rows the judge scored within one point.
	Within1 float64
	// Bias is the mean of judge - human: positive is lenient.
	Bias float64
	// FalsePass is the share of human-failed rows (human < 6) the judge
	// passed as fully correct. This is the error that matters most.
	FalsePass      float64
	FalsePassDenom int
	// FalseFail is the share of human-passed rows (human >= 6) the judge
	// did not pass.
	FalseFail      float64
	FalseFailDenom int
	USD            float64
}

// ReadCalibrationCSV reads a calibration file.
func ReadCalibrationCSV(r io.Reader) ([]CalRow, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	recs, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("empty calibration file")
	}
	col := map[string]int{}
	for i, h := range recs[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, need := range []string{"problem_id", "proof_text", "human_score"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("calibration file needs a %q column", need)
		}
	}
	var rows []CalRow
	for n, rec := range recs[1:] {
		get := func(name string) string {
			i, ok := col[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return rec[i]
		}
		score, err := strconv.Atoi(strings.TrimSpace(get("human_score")))
		if err != nil || score < 0 || score > MaxScore {
			return nil, fmt.Errorf("row %d: human_score must be an integer 0-%d", n+2, MaxScore)
		}
		rows = append(rows, CalRow{ProblemID: strings.TrimSpace(get("problem_id")), Proof: get("proof_text"), Human: score, Label: get("label")})
	}
	return rows, nil
}

// WriteCalibrationCSV writes rows in the calibration format.
func WriteCalibrationCSV(w io.Writer, rows []CalRow) error {
	cw := csv.NewWriter(w)
	cw.Write([]string{"problem_id", "proof_text", "human_score", "label"})
	for _, r := range rows {
		cw.Write([]string{r.ProblemID, r.Proof, strconv.Itoa(r.Human), r.Label})
	}
	cw.Flush()
	return cw.Error()
}

// SampleRows are the bank's synthetic proofs as calibration rows.
func (b *Bank) SampleRows() []CalRow {
	var rows []CalRow
	for _, id := range b.IDs() {
		for _, s := range b.problems[id].Samples {
			rows = append(rows, CalRow{ProblemID: id, Proof: s.Proof, Human: s.Score, Label: s.Kind})
		}
	}
	return rows
}

// Calibrate grades every row, reporting each to progress as it finishes.
func Calibrate(ctx context.Context, g *Grader, rows []CalRow, progress io.Writer) ([]CalResult, CalStats) {
	out := make([]CalResult, 0, len(rows))
	for _, row := range rows {
		res := CalResult{Row: row}
		p := g.Bank.Problem(row.ProblemID)
		if p == nil {
			res.Err = "unknown problem"
		} else {
			o := g.GradeProof(ctx, p, row.Proof)
			res.USD = o.USD
			if o.Unavailable != "" {
				res.Err = o.Unavailable
			} else {
				res.Judge, res.Pass, res.Runs = o.Report.Score, o.Report.FullyCorrect, o.Report.Runs
			}
		}
		out = append(out, res)
		if progress != nil {
			if res.Err != "" {
				fmt.Fprintf(progress, "%-28s %-9s human %d  ungraded: %s\n", row.ProblemID, row.Label, row.Human, res.Err)
			} else {
				fmt.Fprintf(progress, "%-28s %-9s human %d  judge %d  runs %v  pass %v\n", row.ProblemID, row.Label, row.Human, res.Judge, res.Runs, res.Pass)
			}
		}
	}
	return out, Stats(out)
}

// Stats summarises results; ungraded rows are counted but not scored.
func Stats(results []CalResult) CalStats {
	var s CalStats
	var absSum, sum float64
	var within, fp, ff int
	for _, r := range results {
		s.USD += r.USD
		if r.Err != "" {
			s.Ungraded++
			continue
		}
		s.N++
		d := float64(r.Judge - r.Row.Human)
		sum += d
		absSum += math.Abs(d)
		if math.Abs(d) <= 1 {
			within++
		}
		if r.Row.Human < PassMark {
			s.FalsePassDenom++
			if r.Pass {
				fp++
			}
		} else {
			s.FalseFailDenom++
			if !r.Pass {
				ff++
			}
		}
	}
	if s.N > 0 {
		s.MAE = absSum / float64(s.N)
		s.Bias = sum / float64(s.N)
		s.Within1 = float64(within) / float64(s.N)
	}
	if s.FalsePassDenom > 0 {
		s.FalsePass = float64(fp) / float64(s.FalsePassDenom)
	}
	if s.FalseFailDenom > 0 {
		s.FalseFail = float64(ff) / float64(s.FalseFailDenom)
	}
	return s
}

// String is the report printed after a calibration run.
func (s CalStats) String() string {
	return fmt.Sprintf(`graded rows         %d (ungraded %d)
MAE                 %.2f points of 7
within +/-1         %.0f%%
bias (judge-human)  %+.2f  (positive = lenient)
false-pass rate     %.0f%% of %d human-failed proofs passed as fully correct
false-fail rate     %.0f%% of %d human-passed proofs not passed
cost                $%.4f`,
		s.N, s.Ungraded, s.MAE, 100*s.Within1, s.Bias, 100*s.FalsePass, s.FalsePassDenom, 100*s.FalseFail, s.FalseFailDenom, s.USD)
}
