package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/config"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/evaluation"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/model"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/tools"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/tui"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func run(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("dotamanai-codex", flag.ContinueOnError)
	path := f.String("config", "", "TOML config path")
	session := f.String("session", "", "resume session ID")
	mode := f.String("mode", "", "analysis, confirm or auto")
	asJSON := f.Bool("json", false, "JSON output")
	if err := f.Parse(args); err != nil {
		return err
	}
	c, err := config.Load(*path)
	if err != nil {
		return err
	}
	if *mode != "" {
		c.Mode = *mode
		if err = c.Validate(); err != nil {
			return err
		}
	}
	p, err := model.New(c.Endpoint)
	if err != nil {
		return err
	}
	a, err := app.Open(c, p)
	if err != nil {
		return err
	}
	defer a.Close()
	encode := func(v any) error { return json.NewEncoder(os.Stdout).Encode(v) }
	switch f.Arg(0) {
	case "":
		return tui.Run(a, *session)
	case "evaluate":
		v, err := evaluation.Run(ctx, f.Arg(1))
		if err != nil {
			return err
		}
		return encode(v)
	case "doctor":
		if err = p.Health(ctx); err != nil {
			return fmt.Errorf("SQLite ready; local llama-server unavailable: %w", err)
		}
		info, err := p.ModelInfo(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Dot Aman AI · SQLite WAL/FTS5 ready · mode=%s · local model=%s\n", c.Mode, info)
		return nil
	case "summarize":
		messages, err := a.Store.Messages(ctx, f.Arg(1))
		if err != nil {
			return err
		}
		return a.Memory.Compress(ctx, f.Arg(1), messages, p, c.Context)
	case "sessions":
		v, err := a.Store.Sessions(ctx)
		if err != nil {
			return err
		}
		return encode(v)
	case "export-session":
		v, err := a.Store.Messages(ctx, f.Arg(1))
		if err != nil {
			return err
		}
		return encode(v)
	case "delete-session":
		return a.Store.DeleteSession(ctx, f.Arg(1))
	case "import":
		if f.NArg() != 2 {
			return fmt.Errorf("import requires one file path")
		}
		return a.Knowledge.Import(ctx, f.Arg(1))
	case "search":
		v, err := a.Knowledge.Search(ctx, strings.Join(f.Args()[1:], " "), 5)
		if err != nil {
			return err
		}
		return encode(v)
	case "sources":
		v, err := a.Knowledge.Sources(ctx)
		if err != nil {
			return err
		}
		return encode(v)
	case "sync":
		return a.Knowledge.Sync(ctx, f.Arg(1))
	case "chat":
		prompt := strings.Join(f.Args()[1:], " ")
		var stream func(string) error
		if !*asJSON {
			stream = func(s string) error { _, err := fmt.Fprint(os.Stdout, s); return err }
		}
		o, err := a.Run(ctx, *session, prompt, func(tools.Call) bool { return false }, stream)
		if err != nil {
			return err
		}
		if *asJSON {
			return encode(o)
		}
		fmt.Println("\n" + tui.Stats(o))
		fmt.Println("Session:", o.Session)
		return nil
	default:
		return fmt.Errorf("commands: doctor, chat TEXT, sessions, export-session ID, delete-session ID, import PATH, search QUERY, sources, sync cisa-kev|owasp")
	}
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
