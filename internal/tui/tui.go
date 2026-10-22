// Package tui provides a compact terminal conversation with explicit tool confirmation.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/memory"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/reporting"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/tools"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Clipboard interface {
	Copy(context.Context, string) error
}
type WaylandClipboard struct{}

func (WaylandClipboard) Copy(ctx context.Context, s string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wl-copy")
	cmd.Stdin = strings.NewReader(s)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy: %w", err)
	}
	return nil
}

type fragment string
type finished struct {
	out app.Outcome
	err error
}
type proposal struct {
	call  tools.Call
	reply chan bool
}
type Model struct {
	App        *app.App
	Clipboard  Clipboard
	input      textinput.Model
	view       viewport.Model
	transcript string
	status     string
	session    string
	last       app.Outcome
	busy       bool
	cancel     context.CancelFunc
	pending    *proposal
	width      int
	send       func(tea.Msg)
}

func New(a *app.App, session string, clipboard Clipboard) Model {
	in := textinput.New()
	in.Placeholder = "اكتب طلبك أو /help"
	in.CharLimit = 8192
	in.Focus()
	return Model{App: a, Clipboard: clipboard, input: in, view: viewport.New(80, 18), width: 80, session: session, status: "محلي فقط · /help · Ctrl+C إلغاء · Ctrl+Q خروج"}
}
func (m Model) Init() tea.Cmd { return textinput.Blink }
func (m *Model) append(s string) {
	m.transcript += ansi.Strip(s)
	if len(m.transcript) > 131072 {
		r := []rune(m.transcript)
		if len(r) > 32768 {
			m.transcript = string(r[len(r)-32768:])
		}
	}
	m.view.SetContent(ansi.Hardwrap(m.transcript, max(10, m.width-2), true))
	m.view.GotoBottom()
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.view.Width = max(10, v.Width-2)
		m.view.Height = max(3, v.Height-6)
		m.input.Width = max(10, v.Width-4)
		m.view.SetContent(ansi.Hardwrap(m.transcript, max(10, v.Width-2), true))
		return m, nil
	case fragment:
		m.append(string(v))
		return m, nil
	case proposal:
		m.pending = &v
		m.status = "تنفيذ الأداة؟ y موافق / n رفض: " + v.call.Name + " " + string(v.call.Arguments)
		return m, nil
	case finished:
		m.busy = false
		m.pending = nil
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		if v.err != nil {
			m.append("\nخطأ: " + v.err.Error() + "\n")
			m.status = "لم تكتمل الاستجابة"
			if v.out.Session != "" {
				m.session = v.out.Session
			}
			return m, nil
		}
		m.last = v.out
		m.session = v.out.Session
		m.append("\n")
		if renderer, e := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(max(10, m.width-4))); e == nil {
			if rendered, e := renderer.Render(m.transcript); e == nil {
				m.view.SetContent(rendered)
				m.view.GotoBottom()
			}
		}
		m.status = Stats(v.out)
		return m, nil
	case tea.KeyMsg:
		if v.String() == "ctrl+q" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		if v.String() == "ctrl+c" {
			if m.busy && m.cancel != nil {
				m.cancel()
				m.status = "جارٍ الإلغاء"
				return m, nil
			}
			return m, tea.Quit
		}
		if m.pending != nil {
			if v.String() == "y" || v.String() == "n" {
				m.pending.reply <- v.String() == "y"
				m.pending = nil
				m.status = "جارٍ التنفيذ"
			}
			return m, nil
		}
		if v.String() == "enter" && !m.busy {
			text := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if text == "" {
				return m, nil
			}
			if strings.HasPrefix(text, "/") {
				cmd := m.command(text)
				return m, cmd
			}
			m.append("\nأنت: " + text + "\nأمان: ")
			m.busy = true
			m.status = "جارٍ الاستدلال المحلي…"
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			a := m.App
			session := m.session
			send := m.send
			return m, func() tea.Msg {
				approve := func(c tools.Call) bool {
					reply := make(chan bool, 1)
					send(proposal{c, reply})
					select {
					case answer := <-reply:
						return answer
					case <-ctx.Done():
						return false
					}
				}
				out, err := a.Run(ctx, session, text, approve, func(s string) error { send(fragment(s)); return ctx.Err() })
				return finished{out, err}
			}
		}
	}
	var cmd1, cmd2 tea.Cmd
	if !m.busy {
		m.input, cmd1 = m.input.Update(msg)
	}
	m.view, cmd2 = m.view.Update(msg)
	return m, tea.Batch(cmd1, cmd2)
}
func Stats(o app.Outcome) string {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	rate := "N/A"
	if o.Duration > 0 && o.Usage.Output > 0 {
		rate = fmt.Sprintf("%.1f", float64(o.Usage.Output)/o.Duration.Seconds())
	}
	tokens := "N/A"
	if o.Usage.Output > 0 {
		tokens = strconv.Itoa(o.Usage.Output)
	}
	return fmt.Sprintf("Last response · %s tokens · %s tok/s · %.2fs\nApp Go heap %.1f MiB · CPU N/A · GPU N/A · /copy", tokens, rate, o.Duration.Seconds(), float64(mem.Alloc)/(1<<20))
}
func (m Model) View() string {
	return lipgloss.NewStyle().Bold(true).Render("Dot Aman AI · "+m.App.Config.Mode) + "\n" + m.view.View() + "\n" + m.input.View() + "\n" + ansi.Strip(m.status) + "\n"
}

// command is local UI control. Imported content never enters this dispatch path.
func (m *Model) command(text string) tea.Cmd {
	parts := strings.SplitN(text, " ", 2)
	name := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	show := func(v any, err error) {
		if err != nil {
			m.append("\nخطأ: " + err.Error() + "\n")
			return
		}
		b, e := json.MarshalIndent(v, "", "  ")
		if e != nil {
			m.append(e.Error())
			return
		}
		m.append("\n" + string(b) + "\n")
	}
	switch name {
	case "/help":
		m.append("\n/help /new /sessions /resume ID /delete-session ID /mode analysis|confirm|auto\n/search WORDS /import PATH /sources /remember TEXT /facts /forget ID\n/correct ID TEXT /export /stats /copy · PgUp/PgDown scroll\n")
	case "/new":
		m.session = ""
		m.transcript = ""
		m.append("جلسة جديدة\n")
	case "/sessions":
		x, e := m.App.Store.Sessions(ctx)
		show(x, e)
	case "/resume":
		x, e := m.App.Store.Messages(ctx, arg)
		if e != nil {
			show(nil, e)
			break
		}
		sessions, e := m.App.Store.Sessions(ctx)
		if e != nil {
			show(nil, e)
			break
		}
		found := false
		for _, s := range sessions {
			if s.ID == arg {
				found = true
			}
		}
		if !found {
			m.append("\nالجلسة غير موجودة\n")
			break
		}
		m.session = arg
		m.transcript = ""
		for _, v := range x {
			m.append(v.Role + ": " + v.Content + "\n")
		}
	case "/delete-session":
		e := m.App.Store.DeleteSession(ctx, arg)
		show("deleted", e)
		if e == nil && arg == m.session {
			m.session = ""
			m.transcript = ""
			m.append("تم حذف الجلسة\n")
		}
	case "/mode":
		if arg == "analysis" || arg == "confirm" || arg == "auto" {
			m.App.Config.Mode = arg
			m.status = "mode=" + arg
		} else {
			m.status = "invalid mode"
		}
	case "/search":
		x, e := m.App.Knowledge.Search(ctx, arg, 5)
		show(x, e)
	case "/import":
		show("imported", m.App.Knowledge.Import(ctx, arg))
	case "/sources":
		x, e := m.App.Knowledge.Sources(ctx)
		show(x, e)
	case "/remember":
		if m.session == "" {
			m.status = "ابدأ جلسة أولاً"
			break
		}
		id, e := m.App.Memory.Put(ctx, memory.Item{Session: m.session, Kind: "project", Text: arg, Source: "explicit user command", Reviewed: true})
		show(id, e)
	case "/facts":
		x, e := m.App.Memory.Recall(ctx, m.session, "")
		show(x, e)
	case "/forget":
		id, e := strconv.ParseInt(arg, 10, 64)
		if e == nil {
			e = m.App.Memory.Delete(ctx, m.session, id)
		}
		show("deleted", e)
	case "/correct":
		p := strings.SplitN(arg, " ", 2)
		if len(p) != 2 {
			m.status = "/correct ID TEXT"
			break
		}
		id, e := strconv.ParseInt(p[0], 10, 64)
		if e == nil {
			e = m.App.Memory.Correct(ctx, m.session, id, p[1], "explicit user correction")
		}
		show("corrected", e)
	case "/export":
		if m.last.Session == "" {
			m.status = "لا توجد نتيجة لتصديرها"
			break
		}
		p, e := reporting.Export(filepath.Join(m.App.Config.DataDir, "reports"), m.last)
		show(p, e)
	case "/stats":
		m.append("\n" + Stats(m.last) + "\n")
	case "/copy":
		if m.last.Text == "" {
			m.status = "لا توجد نتيجة للنسخ"
			break
		}
		if m.Clipboard == nil {
			m.status = "clipboard unavailable"
			break
		}
		if e := m.Clipboard.Copy(ctx, m.last.Text); e != nil {
			m.status = e.Error()
		} else {
			m.status = "تم نسخ آخر نتيجة"
		}
	default:
		m.status = "أمر غير معروف؛ /help"
	}
	return nil
}
func Run(a *app.App, session string) error {
	var p *tea.Program
	m := New(a, session, WaylandClipboard{})
	m.send = func(msg tea.Msg) { p.Send(msg) }
	p = tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
