package pytrack

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Runner executes learner code in a separate, limited Python process.
//
// One process per submission, rather than a pooled interpreter like the SymPy
// service: a pooled process would carry one learner's globals, open figures
// and monkey-patches into the next run. The cost is interpreter start-up plus
// whatever the code imports (about 0.1 s for NumPy, about 1 s for
// scikit-learn), which is small next to the time a learner spends writing.
type Runner struct {
	Python string // interpreter; PYRUNNER_PYTHON, else "python3"
	Script string // path to grading/pyrunner.py

	// CPUSeconds is the learner-visible time limit. The wall-clock deadline
	// adds start-up and import allowance on top, so a slow container start is
	// not reported as the learner's infinite loop.
	CPUSeconds int
	WallExtra  time.Duration
	MemoryMB   int
	MaxOutput  int

	// DenyPaths are path prefixes learner code may not open (the database
	// directory, the JWT secret). Belt and braces: in the Docker image the
	// process also runs as an unprivileged user.
	DenyPaths []string

	sem       chan struct{}
	mplOnce   sync.Once
	mplDir    string
	availOnce sync.Once
	availErr  error
}

// Request is one execution.
type Request struct {
	Mode    string // "run" or "grade"
	Code    string
	Setup   string
	Tests   []Test
	Figures bool
}

// TestResult is one hidden check.
type TestResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Result is what the runner reported.
type Result struct {
	Stdout          string       `json:"stdout"`
	StdoutTruncated bool         `json:"stdout_truncated"`
	Error           string       `json:"error"`
	SetupError      string       `json:"setup_error"`
	Tests           []TestResult `json:"tests"`
	Passed          bool         `json:"passed"`
	Figures         []string     `json:"figures"`
	TimedOut        bool         `json:"timed_out"`
}

// ErrUnavailable means the code could not be run at all: no interpreter, no
// runner script, a crashed runner, or the queue is full. It is never the
// learner's mistake and callers must not record it as one.
var ErrUnavailable = errors.New("python runner unavailable")

// NewRunner builds a runner from the environment.
func NewRunner() *Runner {
	py := os.Getenv("PYRUNNER_PYTHON")
	if py == "" {
		py = "python3"
	}
	conc := 2
	if v, err := strconv.Atoi(os.Getenv("PYRUNNER_CONCURRENCY")); err == nil && v > 0 {
		conc = v
	}
	cpu := 8
	if v, err := strconv.Atoi(os.Getenv("PYRUNNER_TIME_LIMIT")); err == nil && v > 0 && v <= 60 {
		cpu = v
	}
	return &Runner{
		Python:     py,
		Script:     findRunnerScript(),
		CPUSeconds: cpu,
		WallExtra:  6 * time.Second,
		MemoryMB:   1536,
		MaxOutput:  20000,
		sem:        make(chan struct{}, conc),
	}
}

func findRunnerScript() string {
	candidates := []string{}
	if p := os.Getenv("PYRUNNER_PATH"); p != "" {
		candidates = append(candidates, p)
	}
	rels := []string{"grading/pyrunner.py", "../grading/pyrunner.py", "../../grading/pyrunner.py", "../../../grading/pyrunner.py"}
	candidates = append(candidates, rels...)
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(d, "grading", "pyrunner.py"), filepath.Join(d, "..", "..", "grading", "pyrunner.py"))
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs
		}
	}
	return ""
}

// TimeLimit is the learner-visible limit in seconds.
func (r *Runner) TimeLimit() int { return r.CPUSeconds }

// matplotlibDir returns a shared, pre-warmed MPLCONFIGDIR.
//
// Matplotlib builds a font cache on first import, which takes seconds and
// needs a writable directory. Learner code may not write outside its working
// folder, so the cache is built once here, outside the sandbox, and then read
// by every run. The Docker image sets MPLCONFIGDIR to a cache built at image
// build time and this is skipped.
func (r *Runner) matplotlibDir() string {
	r.mplOnce.Do(func() {
		if d := os.Getenv("MPLCONFIGDIR"); d != "" {
			r.mplDir = d
			return
		}
		d := filepath.Join(os.TempDir(), "mathua-mplconfig")
		if err := os.MkdirAll(d, 0o755); err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, r.Python, "-c", "import matplotlib; matplotlib.use('Agg'); import matplotlib.pyplot")
		cmd.Env = append(baseEnv(), "MPLCONFIGDIR="+d, "HOME="+d)
		_ = cmd.Run() // a failure only means the first plotting run is slower
		_ = os.Chmod(d, 0o755)
		r.mplDir = d
	})
	return r.mplDir
}

// Available reports whether the interpreter can import the data-science stack.
// Checked once; the answer does not change while the server runs.
func (r *Runner) Available() error {
	r.availOnce.Do(func() {
		if r.Script == "" {
			r.availErr = fmt.Errorf("%w: grading/pyrunner.py not found", ErrUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, r.Python, "-I", "-c", "import numpy, pandas, matplotlib, sklearn")
		cmd.Env = baseEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			r.availErr = fmt.Errorf("%w: %s cannot import the data-science libraries: %v %s", ErrUnavailable, r.Python, err, strings.TrimSpace(string(out)))
		}
	})
	return r.availErr
}

func baseEnv() []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"PYTHONIOENCODING=utf-8",
		"MPLBACKEND=Agg",
		"OPENBLAS_NUM_THREADS=1",
		"OMP_NUM_THREADS=1",
		"MKL_NUM_THREADS=1",
		"NUMEXPR_MAX_THREADS=1",
		"JOBLIB_MULTIPROCESSING=0",
		"PYTHONDONTWRITEBYTECODE=1",
	}
	if v := os.Getenv("SYSTEMROOT"); v != "" { // Windows needs it to start Python
		env = append(env, "SYSTEMROOT="+v)
	}
	return env
}

func newNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "PYR" + hex.EncodeToString(b) + ":"
}

// Run executes one request. The returned error is ErrUnavailable (wrapped)
// for infrastructure faults; a learner's exception, failed check or timeout
// is reported in Result, not as an error.
func (r *Runner) Run(ctx context.Context, req Request) (*Result, error) {
	if err := r.Available(); err != nil {
		return nil, err
	}
	select {
	case r.sem <- struct{}{}:
		defer func() { <-r.sem }()
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("%w: busy, try again", ErrUnavailable)
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
	}

	work, err := os.MkdirTemp("", "mathua-run-")
	if err != nil {
		return nil, fmt.Errorf("%w: temp dir: %v", ErrUnavailable, err)
	}
	defer os.RemoveAll(work)
	if err := prepareWorkdir(work); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	nonce := newNonce()
	type wireTest struct {
		Name string `json:"name"`
		Code string `json:"code"`
		Hint string `json:"hint,omitempty"`
	}
	tests := make([]wireTest, 0, len(req.Tests))
	for _, t := range req.Tests {
		tests = append(tests, wireTest{Name: t.Name, Code: t.Code, Hint: t.Hint})
	}
	payload, err := json.Marshal(map[string]interface{}{
		"nonce":       nonce,
		"mode":        req.Mode,
		"code":        req.Code,
		"setup":       req.Setup,
		"tests":       tests,
		"cpu_seconds": r.CPUSeconds,
		"memory_mb":   r.MemoryMB,
		"max_output":  r.MaxOutput,
		"figures":     req.Figures,
		"deny_paths":  r.DenyPaths,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encode: %v", ErrUnavailable, err)
	}

	wall := time.Duration(r.CPUSeconds)*time.Second + r.WallExtra
	runCtx, cancel := context.WithTimeout(ctx, wall)
	defer cancel()

	cmd := exec.CommandContext(runCtx, r.Python, "-I", r.Script)
	cmd.Dir = work
	mpl := r.matplotlibDir()
	cmd.Env = append(baseEnv(), "HOME="+work, "TMPDIR="+work)
	if mpl != "" {
		cmd.Env = append(cmd.Env, "MPLCONFIGDIR="+mpl)
	}
	cmd.Stdin = bytes.NewReader(payload)
	var stdout limitedBuffer
	stdout.limit = 8 << 20
	cmd.Stdout = &stdout
	cmd.Stderr = &limitedBuffer{limit: 64 << 10}
	sandbox(cmd)
	cmd.WaitDelay = 2 * time.Second

	runErr := cmd.Run()
	out := stdout.String()
	if i := strings.Index(out, nonce); i >= 0 {
		line := out[i+len(nonce):]
		if j := strings.IndexByte(line, '\n'); j >= 0 {
			line = line[:j]
		}
		var res Result
		if err := json.Unmarshal([]byte(line), &res); err != nil {
			return nil, fmt.Errorf("%w: bad runner output: %v", ErrUnavailable, err)
		}
		return &res, nil
	}
	// No verdict line. Either the deadline or the CPU limit stopped the
	// process (the learner's code ran too long) or the runner itself failed.
	if runCtx.Err() == context.DeadlineExceeded || killedByLimit(cmd.ProcessState) {
		return &Result{TimedOut: true}, nil
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
	}
	if killedByMemory(cmd.ProcessState) {
		return &Result{Error: "MemoryError: your code used more memory than practice code is allowed."}, nil
	}
	return nil, fmt.Errorf("%w: runner exited without a result (%v)", ErrUnavailable, runErr)
}

type limitedBuffer struct {
	buf   bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	room := b.limit - b.buf.Len()
	if room > 0 {
		if len(p) > room {
			b.buf.Write(p[:room])
		} else {
			b.buf.Write(p)
		}
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string { return b.buf.String() }
