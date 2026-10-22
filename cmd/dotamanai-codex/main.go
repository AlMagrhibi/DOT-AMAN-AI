package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/config"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
	"os"
	"os/signal"
	"syscall"
)

func run(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("dotamanai-codex", flag.ContinueOnError)
	path := f.String("config", "", "TOML config path")
	if err := f.Parse(args); err != nil {
		return err
	}
	c, err := config.Load(*path)
	if err != nil {
		return err
	}
	s, err := storage.Open(c.DataDir)
	if err != nil {
		return err
	}
	defer s.Close()
	switch f.Arg(0) {
	case "sessions":
		v, e := s.Sessions(ctx)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(v)
	case "doctor":
		fmt.Printf("Dot Aman AI · SQLite ready · mode=%s · endpoint=%s\n", c.Mode, c.Endpoint)
		return nil
	default:
		return fmt.Errorf("available commands: doctor, sessions; chat is not implemented yet")
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
