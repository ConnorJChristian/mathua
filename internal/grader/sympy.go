package grader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chuma-beep/mathua/internal/verify"
)

var sympyServicePath string
var findOnce sync.Once

// Pool for persistent sympy service to avoid per-request fork + DoS.
var (
	sympyMu     sync.Mutex
	sympyCmd    *exec.Cmd
	sympyStdin  io.WriteCloser
	sympyStdout *bufio.Scanner
	sympySem    = make(chan struct{}, 4) // limit concurrent grading to 4
)

func findSymPyService() string {
	findOnce.Do(func() {
		paths := []string{
			"grading/sympy_service.py",
			"../../grading/sympy_service.py",
			"../grading/sympy_service.py",
			"../../../grading/sympy_service.py",
		}
		if p := os.Getenv("SYMPY_SERVICE_PATH"); p != "" {
			paths = append([]string{p}, paths...)
		}
		if exe, err := os.Executable(); err == nil {
			d := filepath.Dir(exe)
			paths = append(paths,
				filepath.Join(d, "grading", "sympy_service.py"),
				filepath.Join(d, "..", "..", "grading", "sympy_service.py"),
			)
		}
		if wd, err := os.Getwd(); err == nil {
			for _, rel := range []string{"grading/sympy_service.py", "../../grading/sympy_service.py", "../../../grading/sympy_service.py"} {
				paths = append(paths, filepath.Join(wd, rel))
			}
		}
		for _, p := range paths {
			abs, _ := filepath.Abs(p)
			if _, err := os.Stat(abs); err == nil {
				sympyServicePath = abs
				return
			}
		}
	})
	return sympyServicePath
}

type sympyRequest struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Answer   string `json:"answer"`
}

type sympyResponse struct {
	ID       string `json:"id"`
	Correct  bool   `json:"correct"`
	Feedback string `json:"feedback"`
}

func ensureSympyLocked() error {
	if sympyCmd != nil && sympyCmd.Process != nil {
		// Check if process is still alive (naive: ProcessState == nil means running)
		if sympyCmd.ProcessState == nil {
			return nil
		}
		// Previous process died — clean up
		sympyCmd = nil
		sympyStdin = nil
		sympyStdout = nil
	}
	path := findSymPyService()
	if path == "" {
		return fmt.Errorf("grading service not found")
	}
	cmd := exec.Command("python3", path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("pipe error: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe error: %w", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start error: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	// Bump buffer for 500-char input + JSON overhead; service var limit is 1MB
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	sympyCmd = cmd
	sympyStdin = stdin
	sympyStdout = scanner
	return nil
}

// constEvalNumeric evaluates an expression containing only numbers and
// arithmetic operators (no variables, no functions). `^` is treated as a power.
// Returns false when the expression is not a plain constant.
func constEvalNumeric(expr string) (float64, bool) {
	e := strings.TrimSpace(expr)
	if e == "" {
		return 0, false
	}
	for _, r := range e {
		if !strings.ContainsRune("0123456789.+-*/^() ", r) {
			return 0, false
		}
	}
	e = strings.ReplaceAll(e, "^", "**")
	v, err := verify.Eval(e)
	if err != nil {
		return 0, false
	}
	return v, true
}

// sympySeq numbers requests, so a reply that belongs to an earlier, timed-out
// request is never read as the answer to this one.
var sympySeq uint64

// killSympyLocked stops the service after a fault. The caller holds sympyMu.
func killSympyLocked() {
	if sympyCmd != nil && sympyCmd.Process != nil {
		sympyCmd.Process.Kill()
		sympyCmd.Wait()
	}
	sympyCmd = nil
	sympyStdin = nil
	sympyStdout = nil
}

func gradeSymPy(expected, answer string) Result {
	if strings.TrimSpace(answer) == "" {
		return emptyResult()
	}
	// Fast path: exact match after normalization needs no CAS at all, and
	// guarantees that expected == answer is graded correct even if the SymPy
	// runtime is unavailable (the reported "You: 5^4 · Correct: 5^4" ✗ bug).
	if normaliseSymbolic(expected) == normaliseSymbolic(answer) {
		return Result{Correct: true, Score: 1}
	}
	// Constant numeric expressions are evaluated in Go, so numeric-valued
	// symbolic answers (5^4 vs 625) never depend on the subprocess either.
	if ev, ok := constEvalNumeric(expected); ok {
		if av, ok2 := constEvalNumeric(answer); ok2 {
			if math.Abs(ev-av) < 1e-9 || fractionAsDecimal(expected, answer) {
				return Result{Correct: true, Score: 1}
			}
			return Result{Correct: false, Score: 0, Feedback: "Not equivalent"}
		}
	}

	// Semaphore to bound concurrent grading (avoid fork bomb if pool restarts)
	select {
	case sympySem <- struct{}{}:
		defer func() { <-sympySem }()
	default:
		// If semaphore full, try to acquire with timeout
		ctxSem, cancelSem := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelSem()
		select {
		case sympySem <- struct{}{}:
			defer func() { <-sympySem }()
		case <-ctxSem.Done():
			return Result{Correct: false, Score: 0, Feedback: "Grading service busy", Unavailable: true}
		}
	}

	// The service bounds one grading at 3 s itself (signal.alarm); the rest
	// covers its start-up (importing SymPy) and a loaded machine, where the
	// old 3 s timed out before a trivial reply arrived (fork fix).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure pooled process is alive, guarded by mutex (held across I/O — pool is single stdin/stdout)
	sympyMu.Lock()
	defer sympyMu.Unlock()

	if err := ensureSympyLocked(); err != nil {
		return Result{Correct: false, Score: 0, Feedback: "Grading service not found", Unavailable: true}
	}

	sympySeq++
	req := sympyRequest{
		ID:       strconv.FormatUint(sympySeq, 10),
		Expected: expected,
		Answer:   answer,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return Result{Correct: false, Score: 0, Feedback: "Internal error"}
	}

	// Write with context awareness
	writeDone := make(chan error, 1)
	go func() {
		_, err := sympyStdin.Write(append(data, '\n'))
		writeDone <- err
	}()
	select {
	case err := <-writeDone:
		if err != nil {
			// Process likely died — kill and reset for next call
			killSympyLocked()
			return Result{Correct: false, Score: 0, Feedback: "Grading service write error", Unavailable: true}
		}
	case <-ctx.Done():
		// The reply, if it ever comes, would be read as the next request's
		// (fork fix: under load, a late reply graded the next answer, and
		// an empty answer came back "correct"). Start afresh.
		killSympyLocked()
		return Result{Correct: false, Score: 0, Feedback: "Grading service timed out", Unavailable: true}
	}

	// Read the reply with timeout, skipping any reply to an earlier request.
	var resp sympyResponse
	for {
		scanDone := make(chan bool, 1)
		var line string
		var scanErr bool
		scanner := sympyStdout
		go func() {
			if scanner.Scan() {
				line = scanner.Text()
				scanDone <- true
			} else {
				scanErr = true
				scanDone <- false
			}
		}()
		select {
		case ok := <-scanDone:
			if !ok {
				// Scanner failed — process died
				killSympyLocked()
				if scanErr && ctx.Err() == nil {
					return Result{Correct: false, Score: 0, Feedback: "Grading service response error", Unavailable: true}
				}
				if ctx.Err() != nil {
					return Result{Correct: false, Score: 0, Feedback: "Grading service timed out", Unavailable: true}
				}
				return Result{Correct: false, Score: 0, Feedback: "Grading service error", Unavailable: true}
			}
		case <-ctx.Done():
			killSympyLocked()
			return Result{Correct: false, Score: 0, Feedback: "Grading service timed out", Unavailable: true}
		}
		resp = sympyResponse{}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			killSympyLocked()
			return Result{Correct: false, Score: 0, Feedback: "Grading service response error", Unavailable: true}
		}
		if resp.ID == req.ID {
			break
		}
	}
	if resp.Correct {
		return Result{Correct: true, Score: 1}
	}
	feedback := resp.Feedback
	if feedback == "" {
		feedback = "Incorrect"
	}
	return Result{Correct: false, Score: 0, Feedback: feedback}
}
