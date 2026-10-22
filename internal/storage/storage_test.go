package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistenceIsolationEditDelete(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.NewSession(ctx, "عربي")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.NewSession(ctx, "other")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Add(ctx, a, "user", "قرار قديم"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m, err := s.Messages(ctx, a)
	if err != nil || len(m) != 1 {
		t.Fatalf("%v %v", m, err)
	}
	if err = s.Edit(ctx, b, m[0].ID, "wrong"); err == nil {
		t.Fatal("cross session edit")
	}
	if err = s.Edit(ctx, a, m[0].ID, "قرار مصحح"); err != nil {
		t.Fatal(err)
	}
	m, err = s.Messages(ctx, a)
	if err != nil || m[0].Content != "قرار مصحح" {
		t.Fatal(m, err)
	}
	if err = s.DeleteMessage(ctx, a, m[0].ID); err != nil {
		t.Fatal(err)
	}
	m, err = s.Messages(ctx, a)
	if err != nil || len(m) != 0 {
		t.Fatal(m, err)
	}
	if err = s.Add(ctx, "absent", "user", "bad"); err == nil {
		t.Fatal("missing foreign key")
	}
	if err = s.DeleteSession(ctx, a); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "aman.sqlite"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, err)
	}
}

func TestExperimentMetadataMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aman.sqlite")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE experiments(id INTEGER PRIMARY KEY,session TEXT NOT NULL,prompt TEXT NOT NULL,mode TEXT NOT NULL,status TEXT NOT NULL,report TEXT NOT NULL,error TEXT NOT NULL,created TEXT NOT NULL); PRAGMA user_version=1;`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.NewSession(context.Background(), "migration")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SaveExperimentDetail(context.Background(), id, "prompt", "analysis", "complete", "{}", "", "plan", 2, true, "lesson"); err != nil {
		t.Fatal(err)
	}
	experiments, err := s.Experiments(context.Background(), id)
	if err != nil || len(experiments) != 1 || experiments[0].Attempts != 2 || !experiments[0].Validated || experiments[0].Lessons != "lesson" {
		t.Fatalf("migration metadata missing: %+v err=%v", experiments, err)
	}
}
func TestFTS5(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.DB.Exec(`CREATE VIRTUAL TABLE fts_probe USING fts5(content); INSERT INTO fts_probe VALUES('Linux security');`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.DB.QueryRow(`SELECT count(*) FROM fts_probe WHERE fts_probe MATCH ?`, "security").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
}
