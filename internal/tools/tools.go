// Package tools exposes a fixed set of read-only diagnostics with bounded execution.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Call struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
type Result struct {
	Name     string        `json:"name"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration_ns"`
}
type Auditor interface {
	Audit(context.Context, string, string, string, string) error
}
type Runner interface {
	Run(context.Context, string, []string) (string, error)
}
type Engine struct {
	Root   string
	Audit  Auditor
	Runner Runner
}
type Args struct {
	Path  string `json:"path,omitempty"`
	Limit int    `json:"limit,omitempty"`
}
type Spec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

var descriptions = map[string]string{"read_file": "Read UTF-8 file inside configured root", "list_dir": "List directory inside configured root", "system_info": "Kernel and architecture", "go_version": "Installed Go version", "services": "Local systemd service states", "packages": "Arch installed packages", "network": "Local interface addresses", "journal": "Recent local system journal"}

func Specs() []Spec {
	names := []string{"read_file", "list_dir", "system_info", "go_version", "services", "packages", "network", "journal"}
	out := make([]Spec, 0, len(names))
	for _, n := range names {
		p := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
		if n == "read_file" || n == "list_dir" {
			p["properties"] = map[string]any{"path": map[string]any{"type": "string", "maxLength": 4096}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 65536}}
			p["required"] = []string{"path"}
		}
		out = append(out, Spec{n, descriptions[n], p})
	}
	return out
}
func Validate(c Call) (Args, error) {
	if _, ok := descriptions[c.Name]; !ok {
		return Args{}, fmt.Errorf("unknown tool")
	}
	if len(c.Arguments) > 8192 {
		return Args{}, fmt.Errorf("arguments too large")
	}
	var a Args
	d := json.NewDecoder(bytes.NewReader(c.Arguments))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, err
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return a, fmt.Errorf("trailing JSON")
	}
	if string(bytes.TrimSpace(c.Arguments)) == "null" {
		return a, fmt.Errorf("arguments must be object")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(c.Arguments, &raw); err != nil || raw == nil {
		return a, fmt.Errorf("arguments must be an object")
	}
	if c.Name != "read_file" && c.Name != "list_dir" && len(raw) != 0 {
		return a, fmt.Errorf("tool takes no arguments")
	}
	for k, v := range raw {
		if k != "path" && k != "limit" {
			return a, fmt.Errorf("unknown argument")
		}
		if string(v) == "null" {
			return a, fmt.Errorf("null argument")
		}
	}
	if _, ok := raw["limit"]; ok && a.Limit == 0 {
		return a, fmt.Errorf("limit must be positive")
	}
	if c.Name == "read_file" || c.Name == "list_dir" {
		if a.Path == "" || len(a.Path) > 4096 || filepath.IsAbs(a.Path) || !filepath.IsLocal(a.Path) || strings.ContainsRune(a.Path, 0) {
			return a, fmt.Errorf("path must be relative and confined")
		}
		if a.Limit == 0 {
			a.Limit = 16384
		}
		if a.Limit < 1 || a.Limit > 65536 {
			return a, fmt.Errorf("invalid output limit")
		}
	} else if a.Path != "" || a.Limit != 0 {
		return a, fmt.Errorf("tool takes no arguments")
	}
	return a, nil
}
func (e *Engine) Execute(ctx context.Context, session string, c Call) (Result, error) {
	start := time.Now()
	r := Result{Name: c.Name}
	a, err := Validate(c)
	if err != nil {
		if ae := e.record(ctx, session, c.Name, "rejected", "invalid tool or arguments"); ae != nil {
			return r, ae
		}
		return r, err
	}
	if err = e.record(ctx, session, c.Name, "started", ""); err != nil {
		return r, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if c.Name == "read_file" || c.Name == "list_dir" {
		r.Output, err = e.files(ctx, c.Name, a)
	} else {
		commands := map[string][]string{"system_info": {"uname", "-srmo"}, "go_version": {"go", "version"}, "services": {"systemctl", "list-units", "--type=service", "--no-pager", "--plain"}, "packages": {"pacman", "-Q"}, "network": {"ip", "-brief", "address"}, "journal": {"journalctl", "--no-pager", "-n", "100", "--output=short"}}
		v := commands[c.Name]
		runner := e.Runner
		if runner == nil {
			runner = ProcessRunner{}
		}
		r.Output, err = runner.Run(ctx, v[0], v[1:])
	}
	r.Duration = time.Since(start)
	status := "complete"
	if err != nil {
		status = "failed"
		r.Error = err.Error()
	}
	// Audit stores status, not potentially secret file contents or model-provided arguments.
	if ae := e.record(context.WithoutCancel(ctx), session, c.Name, status, ""); ae != nil {
		return r, errors.Join(err, ae)
	}
	return r, err
}
func (e *Engine) record(ctx context.Context, s, n, status, detail string) error {
	if e.Audit == nil {
		return fmt.Errorf("audit unavailable")
	}
	auditCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return e.Audit.Audit(auditCtx, s, n, status, detail)
}
func (e *Engine) files(ctx context.Context, name string, a Args) (string, error) {
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return "", err
	}
	defer root.Close()
	// OpenRoot confines symlinks; reject nonregular files to avoid devices/FIFOs.
	info, err := root.Stat(a.Path)
	if err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if name == "list_dir" {
		if !info.IsDir() {
			return "", fmt.Errorf("not a directory")
		}
		f, err := root.OpenFile(a.Path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return "", err
		}
		defer f.Close()
		items, err := f.ReadDir(256)
		if err != nil && err != io.EOF {
			return "", err
		}
		var b strings.Builder
		for _, item := range items {
			n := item.Name()
			if b.Len()+len(n)+1 > a.Limit {
				break
			}
			b.WriteString(n)
			b.WriteByte('\n')
		}
		return b.String(), nil
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	f, err := root.OpenFile(a.Path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	b, err := io.ReadAll(io.LimitReader(f, int64(a.Limit)+1))
	if len(b) > a.Limit {
		return string(b[:a.Limit]), fmt.Errorf("output truncated")
	}
	return string(b), err
}

type boundedBuffer struct {
	mu        sync.Mutex
	b         bytes.Buffer
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	left := 65536 - b.b.Len()
	if len(p) > left {
		p = p[:left]
		b.truncated = true
	}
	_, _ = b.b.Write(p)
	return n, nil
}

type ProcessRunner struct{}

func (ProcessRunner) Run(ctx context.Context, name string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	var b boundedBuffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	err := cmd.Run()
	if b.truncated {
		err = errors.Join(err, fmt.Errorf("output truncated"))
	}
	return b.b.String(), err
}
