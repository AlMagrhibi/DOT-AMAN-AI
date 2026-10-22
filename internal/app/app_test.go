package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/config"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/model"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/tools"
	"testing"
)

func TestAutomaticMemoryCompression(t *testing.T) {
	c, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	c.DataDir = t.TempDir()
	c.Root = t.TempDir()
	c.Mode = "analysis"
	a, err := Open(c, model.Mock{Text: "إجابة محفوظة"})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ctx := context.Background()
	var session string
	for i := 0; i < 9; i++ {
		out, runErr := a.Run(ctx, session, fmt.Sprintf("رسالة طويلة رقم %d", i), nil, nil)
		if runErr != nil {
			t.Fatal(runErr)
		}
		session = out.Session
	}
	var summaries int
	if err = a.Store.DB.QueryRowContext(ctx, `SELECT count(*) FROM summaries WHERE session=?`, session).Scan(&summaries); err != nil {
		t.Fatal(err)
	}
	if summaries == 0 {
		t.Fatal("automatic compression did not create a summary")
	}
	messages, err := a.Store.Messages(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 18 {
		t.Fatalf("original messages were not retained: got %d", len(messages))
	}
}

func TestWorkflowPersistenceAndModes(t *testing.T) {
	for _, mode := range []string{"analysis", "confirm", "auto"} {
		t.Run(mode, func(t *testing.T) {
			c, err := config.Default()
			if err != nil {
				t.Fatal(err)
			}
			c.DataDir = t.TempDir()
			c.Root = t.TempDir()
			c.Mode = mode
			p := model.Mock{Text: "نتيجة اختبار", Calls: []tools.Call{{Name: "system_info", Arguments: json.RawMessage(`{}`)}}}
			a, err := Open(c, p)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			out, err := a.Run(ctx, "", "افحص Linux", func(tools.Call) bool { return false }, nil)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "auto" && len(out.Results) != 1 {
				t.Fatal("duplicate diagnostic should execute once", len(out.Results))
			}
			if mode != "auto" && len(out.Results) != 0 {
				t.Fatal("unexpected execution")
			}
			id := out.Session
			if err = a.Close(); err != nil {
				t.Fatal(err)
			}
			a, err = Open(c, p)
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			msgs, err := a.Store.Messages(ctx, id)
			if err != nil || len(msgs) != 2 || msgs[1].Content != "نتيجة اختبار" {
				t.Fatal(msgs, err)
			}
		})
	}
}

func TestFailedInferenceRecordedAsFailed(t *testing.T) {
	c, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	c.DataDir = t.TempDir()
	c.Root = t.TempDir()
	a, err := Open(c, model.Mock{Err: fmt.Errorf("synthetic inference failure")})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	o, err := a.Run(context.Background(), "", "اختبار", nil, nil)
	if err == nil {
		t.Fatal("failure hidden")
	}
	e, err := a.Store.Experiments(context.Background(), o.Session)
	if err != nil || len(e) != 1 || e[0].Status != "failed" || e[0].Error == "" {
		t.Fatal(e, err)
	}
}
