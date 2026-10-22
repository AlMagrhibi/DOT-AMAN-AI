package tui

import (
	"context"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/config"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
	"time"
)

type clip struct{ text string }

func (c *clip) Copy(_ context.Context, s string) error { c.text = s; return nil }
func TestResizeCopyAndUnicode(t *testing.T) {
	c, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	c.DataDir = t.TempDir()
	c.Root = t.TempDir()
	a, err := app.Open(c, model.Mock{})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	clipboard := &clip{}
	m := New(a, "", clipboard)
	m.last = app.Outcome{Text: "أمان Linux"}
	m.command("/copy")
	if clipboard.text != "أمان Linux" || !strings.Contains(m.status, "نسخ") {
		t.Fatal(clipboard, m.status)
	}
	v, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	m = v.(Model)
	m.append("عربي English\x1b]52;c;AAAA\a")
	if !strings.Contains(m.View(), "عربي") || strings.Contains(m.transcript, "\x1b") {
		t.Fatal(m.View())
	}
	m.command("/mode auto")
	if a.Config.Mode != "auto" {
		t.Fatal("mode unchanged")
	}
}

func TestCancelRunningRequest(t *testing.T) {
	c, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	c.DataDir = t.TempDir()
	c.Root = t.TempDir()
	a, err := app.Open(c, model.Mock{})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	m := New(a, "", nil)
	ctx, cancel := context.WithCancel(context.Background())
	m.busy = true
	m.cancel = cancel
	v, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = v.(Model)
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("cancel did not reach running request")
	}
	if !strings.Contains(m.status, "الإلغاء") {
		t.Fatalf("unexpected status: %s", m.status)
	}
}
