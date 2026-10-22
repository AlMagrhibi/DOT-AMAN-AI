// Package storage persists original conversations in private local SQLite files.
package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"time"
)

type Store struct{ DB *sql.DB }
type Message struct {
	ID      int64  `json:"id"`
	Session string `json:"session"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Created string `json:"created"`
}
type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "aman.sqlite")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db}
	if err = s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) migrate() error {
	if _, err := s.DB.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;`); err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > 1 {
		return fmt.Errorf("database schema %d is newer than supported", version)
	}
	if _, err = tx.Exec(`CREATE TABLE IF NOT EXISTS sessions(id TEXT PRIMARY KEY,title TEXT NOT NULL,created TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS messages(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,role TEXT NOT NULL CHECK(role IN ('user','assistant','system')),content TEXT NOT NULL,created TEXT NOT NULL);
 CREATE INDEX IF NOT EXISTS messages_session ON messages(session,id);
 CREATE TABLE IF NOT EXISTS summaries(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,level INTEGER NOT NULL,first_message INTEGER NOT NULL,last_message INTEGER NOT NULL,text TEXT NOT NULL,created TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS experiments(id INTEGER PRIMARY KEY,session TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,prompt TEXT NOT NULL,mode TEXT NOT NULL,status TEXT NOT NULL,report TEXT NOT NULL,error TEXT NOT NULL,plan TEXT NOT NULL DEFAULT '',attempts INTEGER NOT NULL DEFAULT 1,validated INTEGER NOT NULL DEFAULT 0,lessons TEXT NOT NULL DEFAULT '',created TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY,session TEXT,tool TEXT,status TEXT,detail TEXT,created TEXT NOT NULL);
 PRAGMA user_version=1;`); err != nil {
		return err
	}
	for _, column := range []struct{ name, definition string }{{"plan", "TEXT NOT NULL DEFAULT ''"}, {"attempts", "INTEGER NOT NULL DEFAULT 1"}, {"validated", "INTEGER NOT NULL DEFAULT 0"}, {"lessons", "TEXT NOT NULL DEFAULT ''"}} {
		if err = ensureColumnTx(tx, "experiments", column.name, column.definition); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ensureColumnTx(tx *sql.Tx, table, column, definition string) error {
	rows, err := tx.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err = rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) NewSession(ctx context.Context, title string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	_, err := s.DB.ExecContext(ctx, `INSERT INTO sessions VALUES(?,?,?)`, id, title, time.Now().UTC().Format(time.RFC3339Nano))
	return id, err
}
func (s *Store) Sessions(ctx context.Context) ([]Session, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,title FROM sessions ORDER BY created DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var x Session
		if err = rows.Scan(&x.ID, &x.Title); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) Add(ctx context.Context, session, role, content string) error {
	if len(content) > 1<<20 {
		return fmt.Errorf("message too large")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO messages(session,role,content,created) VALUES(?,?,?,?)`, session, role, content, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) Messages(ctx context.Context, id string) ([]Message, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,session,role,content,created FROM messages WHERE session=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.ID, &m.Session, &m.Role, &m.Content, &m.Created); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) Edit(ctx context.Context, session string, id int64, text string) error {
	if len(text) > 1<<20 {
		return fmt.Errorf("message too large")
	}
	return s.mutateMessage(ctx, session, id, `UPDATE messages SET content=? WHERE session=? AND id=?`, text, session, id)
}
func (s *Store) DeleteMessage(ctx context.Context, session string, id int64) error {
	return s.mutateMessage(ctx, session, id, `DELETE FROM messages WHERE session=? AND id=?`, session, id)
}
func (s *Store) mutateMessage(ctx context.Context, session string, id int64, query string, args ...any) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, query, args...)
	if err = changed(r, err); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM summaries WHERE session=?`, session); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) RecentMessages(ctx context.Context, id string, limit int) ([]Message, error) {
	if limit < 1 || limit > 256 {
		return nil, fmt.Errorf("invalid history limit")
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,session,role,content,created FROM (SELECT id,session,role,content,created FROM messages WHERE session=? ORDER BY id DESC LIMIT ?) ORDER BY id`, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.ID, &m.Session, &m.Role, &m.Content, &m.Created); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	r, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id=?`, id)
	return changed(r, err)
}
func changed(r sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("record not found")
	}
	return nil
}
func (s *Store) Audit(ctx context.Context, session, tool, status, detail string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO audit(session,tool,status,detail,created) VALUES(?,?,?,?,?)`, session, tool, status, detail, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

type Experiment struct {
	ID        int64  `json:"id"`
	Session   string `json:"session"`
	Prompt    string `json:"prompt"`
	Mode      string `json:"mode"`
	Status    string `json:"status"`
	Report    string `json:"report"`
	Error     string `json:"error"`
	Plan      string `json:"plan"`
	Attempts  int    `json:"attempts"`
	Validated bool   `json:"validated"`
	Lessons   string `json:"lessons"`
	Created   string `json:"created"`
}

func (s *Store) SaveExperiment(ctx context.Context, session, prompt, mode, status, report, detail string) error {
	return s.SaveExperimentDetail(ctx, session, prompt, mode, status, report, detail, "bounded local plan", 1, status == "complete", "")
}

func (s *Store) SaveExperimentDetail(ctx context.Context, session, prompt, mode, status, report, detail, plan string, attempts int, validated bool, lessons string) error {
	if attempts < 1 || attempts > 32 || len(plan) > 4096 || len(lessons) > 16384 {
		return fmt.Errorf("invalid experiment metadata")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO experiments(session,prompt,mode,status,report,error,plan,attempts,validated,lessons,created) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, session, prompt, mode, status, report, detail, plan, attempts, validated, lessons, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) Experiments(ctx context.Context, session string) ([]Experiment, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,session,prompt,mode,status,report,error,plan,attempts,validated,lessons,created FROM experiments WHERE session=? ORDER BY id DESC LIMIT 20`, session)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Experiment{}
	for rows.Next() {
		var x Experiment
		if err = rows.Scan(&x.ID, &x.Session, &x.Prompt, &x.Mode, &x.Status, &x.Report, &x.Error, &x.Plan, &x.Attempts, &x.Validated, &x.Lessons, &x.Created); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
