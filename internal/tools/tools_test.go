package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type audit struct {
	events []string
	fail   bool
}

func (a *audit) Audit(_ context.Context, _, _, s, _ string) error {
	a.events = append(a.events, s)
	if a.fail {
		return errors.New("audit failed")
	}
	return nil
}

type runner struct{ calls int }

func (r *runner) Run(context.Context, string, []string) (string, error) {
	r.calls++
	return "actual", nil
}
func TestRejectBeforeRunner(t *testing.T) {
	for _, c := range []Call{{"shell", json.RawMessage(`{}`)}, {"system_info", json.RawMessage(`{"path":"x"}`)}, {"read_file", json.RawMessage(`{"path":"../secret"}`)}, {"system_info", json.RawMessage(`{"cmd":"rm"}`)}, {"system_info", json.RawMessage(`{} {}`)}, {"system_info", json.RawMessage(`null`)}} {
		a := &audit{}
		r := &runner{}
		e := Engine{Root: t.TempDir(), Audit: a, Runner: r}
		if _, err := e.Execute(context.Background(), "s", c); err == nil || r.calls != 0 || len(a.events) != 1 || a.events[0] != "rejected" {
			t.Fatalf("%+v calls=%d events=%v", c, r.calls, a.events)
		}
	}
}
func TestAuditFailClosed(t *testing.T) {
	r := &runner{}
	e := Engine{Audit: &audit{fail: true}, Runner: r}
	if _, err := e.Execute(context.Background(), "s", Call{"system_info", json.RawMessage(`{}`)}); err == nil || r.calls != 0 {
		t.Fatal(err, r.calls)
	}
}
func TestConfinedFiles(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	e := Engine{Root: root, Audit: &audit{}}
	if _, err := e.Execute(context.Background(), "s", Call{"read_file", json.RawMessage(`{"path":"escape/secret"}`)}); err == nil {
		t.Fatal("symlink escaped")
	}
	if err := os.WriteFile(filepath.Join(root, "ok"), []byte("أمان AI"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := e.Execute(context.Background(), "s", Call{"read_file", json.RawMessage(`{"path":"ok"}`)})
	if err != nil || r.Output != "أمان AI" {
		t.Fatal(r, err)
	}
}
func TestRealProcessAndCancellation(t *testing.T) {
	r := ProcessRunner{}
	out, err := r.Run(context.Background(), "uname", []string{"-s"})
	if err != nil || !strings.Contains(out, "Linux") {
		t.Fatal(out, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err = r.Run(ctx, "sleep", []string{"5"}); err == nil || time.Since(start) > 2*time.Second {
		t.Fatal(err)
	}
}
func FuzzArguments(f *testing.F) {
	f.Add([]byte(`{"path":"ok"}`))
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = Validate(Call{"read_file", b}) })
}
