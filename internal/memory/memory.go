// Package memory stores explicitly reviewed facts and source-linked summaries.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Memory struct{ DB *sql.DB }
type Item struct {
	ID       int64  `json:"id"`
	Session  string `json:"session"`
	Kind     string `json:"kind"`
	Text     string `json:"text"`
	Source   string `json:"source"`
	Reviewed bool   `json:"reviewed"`
}

func New(db *sql.DB) (*Memory, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS memory(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,kind TEXT NOT NULL,text TEXT NOT NULL,source TEXT NOT NULL,reviewed INTEGER NOT NULL,created TEXT NOT NULL); CREATE TABLE IF NOT EXISTS summaries(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,level INTEGER NOT NULL,first_message INTEGER NOT NULL,last_message INTEGER NOT NULL,text TEXT NOT NULL,created TEXT NOT NULL);`)
	return &Memory{db}, err
}
func (m *Memory) Put(ctx context.Context, x Item) (int64, error) {
	if x.Text == "" || len(x.Text) > 16384 || x.Source == "" {
		return 0, fmt.Errorf("text and source required")
	}
	switch x.Kind {
	case "project", "episodic", "experiment":
	default:
		return 0, fmt.Errorf("invalid memory kind")
	}
	r, err := m.DB.ExecContext(ctx, `INSERT INTO memory(session,kind,text,source,reviewed,created) VALUES(?,?,?,?,?,?)`, x.Session, x.Kind, x.Text, x.Source, x.Reviewed, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}
func (m *Memory) Recall(ctx context.Context, session, query string) ([]Item, error) {
	if len(query) > 1024 {
		return nil, fmt.Errorf("query too long")
	}
	// Project memory is intentionally shared across sessions; episodic and
	// session memory remain isolated to the requested session.
	rows, err := m.DB.QueryContext(ctx, `SELECT id,session,kind,text,source,reviewed FROM memory WHERE reviewed=1 AND (session=? OR kind='project') AND instr(lower(text),lower(?))>0 ORDER BY id DESC LIMIT 20`, session, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var x Item
		if err = rows.Scan(&x.ID, &x.Session, &x.Kind, &x.Text, &x.Source, &x.Reviewed); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (m *Memory) Correct(ctx context.Context, session string, id int64, text, source string) error {
	if text == "" || source == "" || len(text) > 16384 {
		return fmt.Errorf("invalid correction")
	}
	// Project memory is intentionally shared across sessions, so an explicit
	// correction from a new session must be possible. Episodic memory remains
	// session-scoped and cannot be edited through another session.
	r, err := m.DB.ExecContext(ctx, `UPDATE memory SET text=?,source=?,reviewed=1 WHERE id=? AND (session=? OR kind='project')`, text, source, id, session)
	return affected(r, err)
}
func (m *Memory) Delete(ctx context.Context, session string, id int64) error {
	// Match Correct: project facts are shared and explicitly deletable from any
	// session; episodic facts remain isolated.
	r, err := m.DB.ExecContext(ctx, `DELETE FROM memory WHERE id=? AND (session=? OR kind='project')`, id, session)
	return affected(r, err)
}
func affected(r sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("memory not found")
	}
	return nil
}
func (m *Memory) Summary(ctx context.Context, session string, level int, first, last int64, text string) error {
	if level < 1 || level > 10 || first > last || text == "" || len(text) > 32768 {
		return fmt.Errorf("invalid summary")
	}
	var n int
	if err := m.DB.QueryRowContext(ctx, `SELECT count(*) FROM messages WHERE session=? AND id IN (?,?)`, session, first, last).Scan(&n); err != nil {
		return err
	}
	want := 2
	if first == last {
		want = 1
	}
	if n != want {
		return fmt.Errorf("summary source messages missing")
	}
	_, err := m.DB.ExecContext(ctx, `INSERT INTO summaries(session,level,first_message,last_message,text,created) VALUES(?,?,?,?,?,?)`, session, level, first, last, text, time.Now().UTC().Format(time.RFC3339))
	return err
}
