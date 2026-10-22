// Package knowledge indexes explicitly imported documents with source provenance.
package knowledge

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Index struct{ DB *sql.DB }
type Document struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Published string `json:"published,omitempty"`
}
type Hit struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Text    string `json:"text"`
	SHA256  string `json:"sha256"`
	Fetched string `json:"fetched"`
}
type Source struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	License  string `json:"license"`
	LastSync string `json:"last_sync"`
	Status   string `json:"status"`
	Error    string `json:"error"`
}

func New(db *sql.DB) (*Index, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sources(id TEXT PRIMARY KEY,url TEXT NOT NULL,license TEXT NOT NULL,last_sync TEXT NOT NULL DEFAULT '',status TEXT NOT NULL DEFAULT 'registered',error TEXT NOT NULL DEFAULT '');
 CREATE TABLE IF NOT EXISTS documents(id TEXT PRIMARY KEY,source TEXT NOT NULL,url TEXT NOT NULL,title TEXT NOT NULL,text TEXT NOT NULL,sha TEXT NOT NULL,fetched TEXT NOT NULL,published TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS document_versions(document TEXT,sha TEXT,text TEXT,fetched TEXT,PRIMARY KEY(document,sha));
 CREATE TABLE IF NOT EXISTS chunks(id INTEGER PRIMARY KEY,document TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,text TEXT NOT NULL);
 CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(normalized,chunk UNINDEXED);
 CREATE TABLE IF NOT EXISTS sync_history(id INTEGER PRIMARY KEY,source TEXT,status TEXT,detail TEXT,created TEXT);
 INSERT OR IGNORE INTO sources(id,url,license) VALUES('cisa-kev','https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json','US government public information; verify third-party notices'),('owasp','https://raw.githubusercontent.com/OWASP/CheatSheetSeries/master/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.md','CC-BY-SA-4.0');`)
	return &Index{db}, err
}
func Normalize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) || r == 'ـ' {
			return -1
		}
		switch r {
		case 'أ', 'إ', 'آ', 'ٱ':
			return 'ا'
		case 'ى':
			return 'ي'
		case 'ؤ':
			return 'و'
		case 'ئ':
			return 'ي'
		}
		return unicode.ToLower(r)
	}, s)
}
func (i *Index) Put(ctx context.Context, d Document) error {
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = i.put(ctx, tx, d); err != nil {
		return err
	}
	return tx.Commit()
}
func (i *Index) put(ctx context.Context, tx *sql.Tx, d Document) error {
	if d.ID == "" || d.Source == "" || d.Title == "" || d.Text == "" || len(d.Text) > 2<<20 || !utf8.ValidString(d.Text) {
		return fmt.Errorf("invalid document")
	}
	sum := sha256.Sum256([]byte(d.Text))
	sha := hex.EncodeToString(sum[:])
	now := time.Now().UTC().Format(time.RFC3339)
	var old string
	err := tx.QueryRowContext(ctx, `SELECT sha FROM documents WHERE id=?`, d.ID).Scan(&old)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if old == sha {
		return nil
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM chunks_fts WHERE chunk IN (SELECT id FROM chunks WHERE document=?)`, d.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM chunks WHERE document=?`, d.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO documents VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET source=excluded.source,url=excluded.url,title=excluded.title,text=excluded.text,sha=excluded.sha,fetched=excluded.fetched,published=excluded.published`, d.ID, d.Source, d.URL, d.Title, d.Text, sha, now, d.Published); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO document_versions VALUES(?,?,?,?)`, d.ID, sha, d.Text, now); err != nil {
		return err
	}
	runes := []rune(d.Text)
	for n := 0; n < len(runes); n += 1000 {
		end := n + 1200
		if end > len(runes) {
			end = len(runes)
		}
		text := string(runes[n:end])
		r, err := tx.ExecContext(ctx, `INSERT INTO chunks(document,text) VALUES(?,?)`, d.ID, text)
		if err != nil {
			return err
		}
		id, err := r.LastInsertId()
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO chunks_fts(normalized,chunk) VALUES(?,?)`, Normalize(d.Title+" "+text), id); err != nil {
			return err
		}
		if end == len(runes) {
			break
		}
	}
	return nil
}
func (i *Index) Search(ctx context.Context, query string, limit int) ([]Hit, error) {
	if limit < 1 || limit > 20 {
		return nil, fmt.Errorf("invalid result limit")
	}
	if len(query) > 4096 {
		return nil, fmt.Errorf("query too long")
	}
	words := strings.FieldsFunc(Normalize(query), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if len(words) == 0 {
		return []Hit{}, nil
	}
	if len(words) > 32 {
		return nil, fmt.Errorf("too many query terms")
	}
	for n, w := range words {
		words[n] = "\"" + w + "\""
	}
	q := strings.Join(words, " AND ")
	rows, err := i.DB.QueryContext(ctx, `SELECT d.id,d.source,d.url,d.title,c.text,d.sha,d.fetched FROM chunks_fts f JOIN chunks c ON c.id=f.chunk JOIN documents d ON d.id=c.document WHERE chunks_fts MATCH ? ORDER BY bm25(chunks_fts) LIMIT ?`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Hit{}
	for rows.Next() {
		var h Hit
		if err = rows.Scan(&h.ID, &h.Source, &h.URL, &h.Title, &h.Text, &h.SHA256, &h.Fetched); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
func (i *Index) Delete(ctx context.Context, id string) error {
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{`DELETE FROM chunks_fts WHERE chunk IN (SELECT id FROM chunks WHERE document=?)`, `DELETE FROM chunks WHERE document=?`, `DELETE FROM document_versions WHERE document=?`, `DELETE FROM documents WHERE id=?`} {
		if _, err = tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (i *Index) Import(ctx context.Context, path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".json", ".csv", ".log", ".toml", ".go", ".sh":
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("regular file required")
	}
	b, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		return err
	}
	if len(b) > 2<<20 || !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
		return fmt.Errorf("invalid or oversized UTF-8 file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(abs))
	return i.Put(ctx, Document{ID: "local:" + hex.EncodeToString(sum[:]), Source: "user-import", URL: "local:" + filepath.Base(path), Title: filepath.Base(path), Text: string(b)})
}
func (i *Index) Sources(ctx context.Context) ([]Source, error) {
	rows, err := i.DB.QueryContext(ctx, `SELECT id,url,license,last_sync,status,error FROM sources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		var s Source
		if err = rows.Scan(&s.ID, &s.URL, &s.License, &s.LastSync, &s.Status, &s.Error); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Sync fetches only built-in immutable HTTPS destinations, never user-provided URLs.
func (i *Index) Sync(ctx context.Context, id string) error {
	urls := map[string]string{"cisa-kev": "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json", "owasp": "https://raw.githubusercontent.com/OWASP/CheatSheetSeries/master/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.md"}
	url, ok := urls[id]
	if !ok {
		return fmt.Errorf("unknown source")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("source redirect rejected") }}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return i.syncError(ctx, id, err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return i.syncError(ctx, id, fmt.Errorf("HTTP %d", res.StatusCode))
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil {
		return i.syncError(ctx, id, err)
	}
	if len(b) > 8<<20 {
		return i.syncError(ctx, id, fmt.Errorf("source too large"))
	}
	docs := []Document{}
	if id == "cisa-kev" {
		var feed struct {
			Vulnerabilities []struct {
				CVE         string `json:"cveID"`
				Name        string `json:"vulnerabilityName"`
				Description string `json:"shortDescription"`
				Action      string `json:"requiredAction"`
				Date        string `json:"dateAdded"`
			} `json:"vulnerabilities"`
		}
		if err = json.Unmarshal(b, &feed); err != nil {
			return i.syncError(ctx, id, err)
		}
		if len(feed.Vulnerabilities) == 0 || len(feed.Vulnerabilities) > 10000 {
			return i.syncError(ctx, id, fmt.Errorf("invalid KEV count"))
		}
		for _, v := range feed.Vulnerabilities {
			if !strings.HasPrefix(v.CVE, "CVE-") || v.Name == "" {
				return i.syncError(ctx, id, fmt.Errorf("invalid KEV record"))
			}
			docs = append(docs, Document{ID: v.CVE, Source: id, URL: url, Title: v.CVE + " " + v.Name, Text: v.CVE + "\n" + v.Description + "\nRequired action: " + v.Action, Published: v.Date})
		}
	} else {
		if !utf8.Valid(b) || len(b) == 0 || len(b) > 2<<20 {
			return i.syncError(ctx, id, fmt.Errorf("invalid OWASP text"))
		}
		docs = append(docs, Document{ID: "owasp-sql-injection", Source: id, URL: url, Title: "OWASP SQL Injection Prevention", Text: string(b)})
	}
	// Commit the entire validated feed and its sync metadata atomically.
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, d := range docs {
		if err = i.put(ctx, tx, d); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE sources SET last_sync=?,status='synced',error='' WHERE id=?`, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO sync_history(source,status,detail,created) VALUES(?,?,?,?)`, id, "synced", fmt.Sprint(len(docs)), time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}
func (i *Index) syncError(ctx context.Context, id string, cause error) error {
	_, err := i.DB.ExecContext(ctx, `UPDATE sources SET status='error',error=? WHERE id=?`, cause.Error(), id)
	if err != nil {
		return fmt.Errorf("sync: %w; record error: %v", cause, err)
	}
	return cause
}
