package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	defaultTimeout = 120 * time.Second
	maxOutputBytes = 2 << 20
)

type CommandAction struct {
	Command string `json:"command"`
}

type CommandExecError struct {
	ExitCode int
	Output   string
}

func (e *CommandExecError) Error() string {
	return fmt.Sprintf("command failed with exit code %d", e.ExitCode)
}

func executeCommandAction(action ToolTask) (string, error) {
	h, ok := commandDispatch[action.Key]
	if !ok {
		log.Printf("❌ Unknown tool key: %s\n", action.Key)
		return "", fmt.Errorf("unknown tool key: %s", action.Key)
	}
	return h(action.Parameters)
}

var commandDispatch = map[string]func(any) (string, error){
	run_go_command: func(p any) (string, error) {
		return withParsed[CommandAction](p, run_go_command, func(ca CommandAction) (string, error) {
			return runGoCMD(ca)
		})
	},
}

func runGoCMD(ca CommandAction) (string, error) {
	cmdline := strings.TrimSpace(ca.Command)
	if cmdline == "" {
		return "", errors.New("command is required")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	if err := rejectDangerousSyntax(cmdline); err != nil {
		return "", err
	}

	tokens, err := tokenizeStrict(cmdline)
	if err != nil {
		return "", err
	}
	if len(tokens) < 2 || tokens[0] != "go" {
		return "", errors.New(`only "go" commands are allowed`)
	}

	mode, args, err := validateAllowed(tokens)
	if err != nil {
		return "", err
	}

	if _, err = exec.LookPath("go"); err != nil {
		return "", errors.New(`"go" binary not found in PATH`)
	}

	root, err := getRoot()
	if err != nil || root == "" {
		return "", errors.New("sandbox root not configured (WORKER_FOLDER)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := exec.CommandContext(ctx, "go", args...)
	c.Dir = root

	cw := newCappedWriter(maxOutputBytes)
	c.Stdout = cw
	c.Stderr = cw

	if err = c.Start(); err != nil {
		return "", err
	}
	waitErr := c.Wait()

	out := cw.String()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("command timed out after %s", defaultTimeout)
	}
	if waitErr != nil {
		var ee *exec.ExitError
		if errors.As(waitErr, &ee) {
			return out, &CommandExecError{ExitCode: exitCodeFromError(ee), Output: out + "\n" + waitErr.Error()}
		}
		return out, waitErr
	}

	log.Printf("✅ go %s executed successfully.\n", mode)
	return out, nil
}

func validateAllowed(tokens []string) (string, []string, error) {
	switch tokens[1] {
	case "test":
		args, err := validateGoTest(tokens[2:])
		if err != nil {
			return "", nil, err
		}
		return "test", append([]string{"test"}, args...), nil
	case "fmt":
		args, err := validateGoFmt(tokens[2:])
		if err != nil {
			return "", nil, err
		}
		return "fmt", append([]string{"fmt"}, args...), nil
	case "mod":
		args, err := validateGoMod(tokens[2:])
		if err != nil {
			return "", nil, err
		}
		return "mod_tidy", append([]string{"mod"}, args...), nil
	default:
		return "", nil, fmt.Errorf("subcommand %q not allowed", tokens[1])
	}
}

func validateGoTest(rest []string) ([]string, error) {
	boolFlags := map[string]struct{}{
		"-v": {}, "-race": {}, "-cover": {}, "-benchmem": {}, "-short": {}, "-failfast": {}, "-json": {},
	}
	valueFlags := map[string]func(string) bool{
		"-run":          isSafeRegex,
		"-bench":        isSafeRegex,
		"-timeout":      isDuration,
		"-tags":         isTagList,
		"-covermode":    isOneOf("set", "count", "atomic"),
		"-coverprofile": isSafeRelPath,
		"-count":        isDigits,
		"-shuffle":      isShuffle,
		"-vet":          isOneOf("off"),
	}
	return validateFlagsAndPackages(rest, boolFlags, valueFlags, true)
}

func validateGoFmt(rest []string) ([]string, error) {
	boolFlags := map[string]struct{}{"-n": {}, "-x": {}}
	return validateFlagsAndPackages(rest, boolFlags, nil, true)
}

func validateGoMod(rest []string) ([]string, error) {
	if len(rest) == 0 || rest[0] != "tidy" {
		return nil, errors.New(`only "go mod tidy" is allowed`)
	}
	allowed := map[string]struct{}{
		"-e": {},
		"-v": {},
	}
	var valFlags map[string]func(string) bool = nil

	args, err := validateFlagsAndPackages(rest[1:], allowed, valFlags, false)

	if err != nil {
		return nil, err
	}
	return append([]string{"tidy"}, args...), nil
}

func validateFlagsAndPackages(rest []string, boolFlags map[string]struct{}, valFlags map[string]func(string) bool, allowPkgs bool) ([]string, error) {
	var args []string
	for i := 0; i < len(rest); i++ {
		t := rest[i]
		if strings.HasPrefix(t, "-") {
			base, val, hasEq := splitFlag(t)
			if _, ok := boolFlags[base]; ok {
				if hasEq {
					return nil, fmt.Errorf("flag %q does not take a value", base)
				}
				args = append(args, base)
				continue
			}
			if valFlags != nil {
				if validate, ok := valFlags[base]; ok {
					if !hasEq {
						return nil, fmt.Errorf("flag %q requires -flag=value form", base)
					}
					if !validate(val) {
						return nil, fmt.Errorf("invalid value for %s", base)
					}
					args = append(args, t)
					continue
				}
			}
			return nil, fmt.Errorf("flag %q not allowed", base)
		}
		if !allowPkgs {
			return nil, fmt.Errorf("unexpected argument: %q", t)
		}
		if !isSafePackageToken(t) {
			return nil, fmt.Errorf("invalid package pattern: %q", t)
		}
		args = append(args, t)
	}
	if len(args) == 0 && allowPkgs {
		args = append(args, "./...")
	}
	return args, nil
}

func splitFlag(s string) (base, val string, hasEq bool) {
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", false
}

func rejectDangerousSyntax(s string) error {
	forbidden := []string{"&&", "||", ";", "|", "\n", "\r", "`", "$(", "<", ">", ">>", "<<", "&"}
	for _, f := range forbidden {
		if strings.Contains(s, f) {
			return fmt.Errorf("forbidden operator detected: %q", f)
		}
	}
	if strings.ContainsAny(s, `"'`) {
		return errors.New("quotes are not allowed")
	}
	return nil
}

func tokenizeStrict(s string) ([]string, error) {
	if strings.ContainsAny(s, `"'`) {
		return nil, errors.New("quotes are not allowed")
	}
	f := strings.Fields(s)
	if len(f) == 0 {
		return nil, errors.New("empty command")
	}
	return f, nil
}

var (
	regexRunBench = regexp.MustCompile(`^[A-Za-z0-9_.$^*+?|\[\]{}\-]+$`)
	durRe         = regexp.MustCompile(`^[0-9]+(ns|us|µs|ms|s|m|h)$`)
	tagsRe        = regexp.MustCompile(`^[A-Za-z0-9_,]+$`)
	digitsRe      = regexp.MustCompile(`^[0-9]+$`)
	shuffleRe     = regexp.MustCompile(`^(on|off|[0-9]+)$`)
	relPathRe     = regexp.MustCompile(`^[A-Za-z0-9._/\-]+$`)
	pkgRe         = regexp.MustCompile(`^(\./)?[A-Za-z0-9._/\-]+(\.\.\.)?$`)
)

func isSafeRegex(v string) bool {
	return v != "" && regexRunBench.MatchString(v)
}

func isDuration(v string) bool {
	return durRe.MatchString(v)
}

func isTagList(v string) bool {
	return tagsRe.MatchString(v)
}

func isDigits(v string) bool {
	return digitsRe.MatchString(v)
}

func isShuffle(v string) bool {
	return shuffleRe.MatchString(v)
}

func isOneOf(vals ...string) func(string) bool {
	m := map[string]struct{}{}
	for _, s := range vals {
		m[s] = struct{}{}
	}
	return func(v string) bool { _, ok := m[v]; return ok }
}

func isSafeRelPath(v string) bool {
	if strings.HasPrefix(v, "/") {
		return false
	}
	if windowsAbsPathLike(v) {
		return false
	}
	if strings.Contains(v, "..") {
		return false
	}
	return relPathRe.MatchString(v)
}

func isSafePackageToken(t string) bool {
	if strings.HasPrefix(t, "/") {
		return false
	}
	if windowsAbsPathLike(t) {
		return false
	}
	if strings.ContainsAny(t, `"'`+"\n\r`$()<>|;&") {
		return false
	}
	if strings.Contains(t, "..") && !strings.Contains(t, "...") {
		return false
	}
	if t == "." || t == "all" {
		return true
	}
	return pkgRe.MatchString(t)
}

func windowsAbsPathLike(s string) bool {
	if len(s) >= 3 && ((s[1] == ':' && (s[2] == '\\' || s[2] == '/')) || strings.HasPrefix(s, `\\`)) {
		return true
	}
	return false
}

type cappedWriter struct {
	buf bytes.Buffer
	max int64
	n   int64
}

func newCappedWriter(max int64) *cappedWriter {
	return &cappedWriter{max: max}
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	remain := w.max - w.n
	if remain <= 0 {
		return len(p), nil
	}
	if int64(len(p)) > remain {
		p = p[:remain]
	}
	n, _ := w.buf.Write(p)
	w.n += int64(n)
	return len(p), nil
}

func (w *cappedWriter) String() string {
	return w.buf.String()
}

func exitCodeFromError(e *exec.ExitError) int {
	if st, ok := e.Sys().(interface{ ExitStatus() int }); ok {
		return st.ExitStatus()
	}
	return -1
}
