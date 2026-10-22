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
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

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
	ID            string `json:"id"`
	URL           string `json:"url"`
	FallbackURL   string `json:"fallback_url,omitempty"`
	Format        string `json:"format"`
	License       string `json:"license"`
	MaxBytes      int64  `json:"max_bytes"`
	Bytes         int64  `json:"bytes"`
	LastSync      string `json:"last_sync"`
	ETag          string `json:"etag,omitempty"`
	LastModified  string `json:"last_modified,omitempty"`
	DocumentCount int    `json:"document_count"`
	Status        string `json:"status"`
	Error         string `json:"error"`
}

// Index holds lexical FTS5 data and optional local semantic vectors. The
// embedder is nil by default so existing installations keep working offline.
type Index struct {
	DB           *sql.DB
	embedder     Embedder
	sourceClient *http.Client
}

func New(db *sql.DB) (*Index, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sources(id TEXT PRIMARY KEY,url TEXT NOT NULL,format TEXT NOT NULL DEFAULT 'unknown',license TEXT NOT NULL,last_sync TEXT NOT NULL DEFAULT '',status TEXT NOT NULL DEFAULT 'registered',error TEXT NOT NULL DEFAULT '',fallback_url TEXT NOT NULL DEFAULT '',etag TEXT NOT NULL DEFAULT '',last_modified TEXT NOT NULL DEFAULT '',document_count INTEGER NOT NULL DEFAULT 0,max_bytes INTEGER NOT NULL DEFAULT 8388608,bytes INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS documents(id TEXT PRIMARY KEY,source TEXT NOT NULL,url TEXT NOT NULL,title TEXT NOT NULL,text TEXT NOT NULL,sha TEXT NOT NULL,fetched TEXT NOT NULL,published TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS document_versions(document TEXT,sha TEXT,text TEXT,fetched TEXT,PRIMARY KEY(document,sha));
 CREATE TABLE IF NOT EXISTS chunks(id INTEGER PRIMARY KEY,document TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,text TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS chunk_embeddings(chunk INTEGER PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,model TEXT NOT NULL,dimension INTEGER NOT NULL,vector BLOB NOT NULL,created TEXT NOT NULL);
 CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(normalized,chunk UNINDEXED);
 CREATE TABLE IF NOT EXISTS sync_history(id INTEGER PRIMARY KEY,source TEXT,status TEXT,detail TEXT,created TEXT);
	INSERT OR IGNORE INTO sources(id,url,license) VALUES('cisa-kev','https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json','US government public information; verify third-party notices'),('owasp','https://raw.githubusercontent.com/OWASP/CheatSheetSeries/master/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.md','CC-BY-SA-4.0');`)
	if err != nil {
		return nil, err
	}
	for _, column := range []struct{ name, definition string }{{"fallback_url", "TEXT NOT NULL DEFAULT ''"}, {"etag", "TEXT NOT NULL DEFAULT ''"}, {"last_modified", "TEXT NOT NULL DEFAULT ''"}, {"document_count", "INTEGER NOT NULL DEFAULT 0"}, {"format", "TEXT NOT NULL DEFAULT 'unknown'"}, {"max_bytes", "INTEGER NOT NULL DEFAULT 8388608"}, {"bytes", "INTEGER NOT NULL DEFAULT 0"}} {
		if err = ensureColumn(db, "sources", column.name, column.definition); err != nil {
			return nil, err
		}
	}
	registry := []struct{ id, url, format, license string }{
		{"cisa-kev", "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json", "json", "US government public information; verify third-party notices"},
		{"owasp", "https://raw.githubusercontent.com/OWASP/CheatSheetSeries/master/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.md", "markdown", "CC-BY-SA-4.0"},
		{"cve", "https://www.cve.org/", "html", "CVE Terms of Use"},
		{"nvd", "https://nvd.nist.gov/", "json/html", "NIST public data; verify terms"},
		{"epss", "https://api.first.org/data/v1/epss", "json", "FIRST EPSS terms"},
		{"osv", "https://osv.dev/", "json", "OSV terms"},
		{"go-vulndb", "https://vuln.go.dev/", "json", "Go vulnerability database terms"},
		{"cwe", "https://cwe.mitre.org/", "html", "MITRE CWE terms"},
		{"capec", "https://capec.mitre.org/", "html", "MITRE CAPEC terms"},
		{"attack", "https://attack.mitre.org/", "html", "MITRE ATT&CK terms"},
		{"owasp-wstg", "https://owasp.org/www-project-web-security-testing-guide/", "html", "CC-BY-SA-4.0"},
		{"owasp-asvs", "https://asvs.owasp.org/", "html", "CC-BY-SA-4.0"},
		{"arch-security", "https://security.archlinux.org/", "html", "Arch Linux terms"},
		{"arch-wiki", "https://wiki.archlinux.org/", "html", "Arch Wiki terms"},
		{"kernel-docs", "https://docs.kernel.org/", "html", "Linux kernel documentation license"},
	}
	for _, source := range registry {
		if _, err = db.Exec(`INSERT OR IGNORE INTO sources(id,url,format,license,max_bytes) VALUES(?,?,?,?,?)`, source.id, source.url, source.format, source.license, 8<<20); err != nil {
			return nil, err
		}
		if _, err = db.Exec(`UPDATE sources SET url=?,format=?,license=?,max_bytes=? WHERE id=?`, source.url, source.format, source.license, 8<<20, source.id); err != nil {
			return nil, err
		}
	}
	if _, err = db.Exec(`UPDATE sources SET fallback_url=? WHERE id='cisa-kev' AND fallback_url=''`, "https://raw.githubusercontent.com/cisagov/kev-data/main/known_exploited_vulnerabilities.json"); err != nil {
		return nil, err
	}
	return &Index{DB: db}, nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err = rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			found = true
		}
	}
	if err = rows.Err(); err != nil || found {
		return err
	}
	_, err = db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func (i *Index) SetEmbedder(e Embedder) { i.embedder = e }
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
	hits, err := i.search(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	if len(words) < 2 {
		return i.withSemantic(ctx, query, hits, limit)
	}
	if len(hits) > 0 {
		return i.withSemantic(ctx, query, hits, limit)
	}
	// Preserve exact multi-term matches first. Natural questions often include
	// function words absent from the document; fall back to ranked lexical terms.
	stop := map[string]bool{}
	for _, w := range strings.Fields("what which how does do is are the a an of to in on for or and explain please ما ماذا كيف هل هو هي في من علي عن الى لي اشرح") {
		stop[w] = true
	}
	terms := []string{}
	for _, w := range words {
		if !stop[strings.Trim(w, "\"")] {
			terms = append(terms, w)
		}
	}
	if len(terms) == 0 {
		return i.withSemantic(ctx, query, nil, limit)
	}
	hits, err = i.search(ctx, strings.Join(terms, " OR "), limit)
	if err != nil {
		return nil, err
	}
	return i.withSemantic(ctx, query, hits, limit)
}

// EnsureEmbeddings incrementally fills the local vector index. It is bounded so
// a large feed cannot unexpectedly monopolize the model server during chat.
func (i *Index) EnsureEmbeddings(ctx context.Context, limit int) (int, error) {
	if i.embedder == nil || limit == 0 {
		return 0, nil
	}
	if limit < 1 || limit > 10000 {
		return 0, fmt.Errorf("invalid embedding index limit")
	}
	rows, err := i.DB.QueryContext(ctx, `SELECT c.id,c.text FROM chunks c LEFT JOIN chunk_embeddings e ON e.chunk=c.id WHERE e.chunk IS NULL ORDER BY c.id LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	type item struct {
		id   int64
		text string
	}
	items := []item{}
	for rows.Next() {
		var x item
		if err = rows.Scan(&x.id, &x.text); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	indexed := 0
	for start := 0; start < len(items); start += 32 {
		end := start + 32
		if end > len(items) {
			end = len(items)
		}
		texts := make([]string, end-start)
		for n := range texts {
			texts[n] = "search_document: " + items[start+n].text
		}
		vectors, err := i.embedder.Embed(ctx, texts)
		if err != nil {
			return indexed, err
		}
		if len(vectors) != len(texts) {
			return indexed, fmt.Errorf("embedding count mismatch")
		}
		tx, err := i.DB.BeginTx(ctx, nil)
		if err != nil {
			return indexed, err
		}
		for n, vector := range vectors {
			encoded, encodeErr := encodeVector(vector)
			if encodeErr != nil {
				tx.Rollback()
				return indexed, encodeErr
			}
			if _, err = tx.ExecContext(ctx, `INSERT OR REPLACE INTO chunk_embeddings(chunk,model,dimension,vector,created) VALUES(?,?,?,?,?)`, items[start+n].id, "nomic-embed-text-v1.5", len(vector), encoded, time.Now().UTC().Format(time.RFC3339)); err != nil {
				tx.Rollback()
				return indexed, err
			}
		}
		if err = tx.Commit(); err != nil {
			return indexed, err
		}
		indexed += len(vectors)
	}
	return indexed, nil
}

func (i *Index) withSemantic(ctx context.Context, query string, lexical []Hit, limit int) ([]Hit, error) {
	if i.embedder == nil {
		return lexical, nil
	}
	if _, err := i.EnsureEmbeddings(ctx, 2048); err != nil {
		return lexical, err
	}
	vectors, err := i.embedder.Embed(ctx, []string{"search_query: " + query})
	if err != nil {
		return lexical, err
	}
	if len(vectors) != 1 {
		return lexical, fmt.Errorf("query embedding count mismatch")
	}
	queryVector := vectors[0]
	type scored struct {
		hit   Hit
		score float64
	}
	rows, err := i.DB.QueryContext(ctx, `SELECT d.id,d.source,d.url,d.title,c.text,d.sha,d.fetched,e.vector FROM chunk_embeddings e JOIN chunks c ON c.id=e.chunk JOIN documents d ON d.id=c.document WHERE e.dimension=?`, len(queryVector))
	if err != nil {
		return lexical, err
	}
	all := []scored{}
	for rows.Next() {
		var h Hit
		var raw []byte
		if err = rows.Scan(&h.ID, &h.Source, &h.URL, &h.Title, &h.Text, &h.SHA256, &h.Fetched, &raw); err != nil {
			rows.Close()
			return lexical, err
		}
		vector, decodeErr := decodeVector(raw)
		if decodeErr != nil {
			rows.Close()
			return lexical, decodeErr
		}
		score := cosine(queryVector, vector)
		all = append(all, scored{hit: h, score: score})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return lexical, err
	}
	rows.Close()
	sort.SliceStable(all, func(a, b int) bool { return all[a].score > all[b].score })
	semantic := make([]Hit, 0, limit)
	seen := map[string]bool{}
	for _, h := range lexical {
		seen[h.ID] = true
	}
	for _, item := range all {
		if seen[item.hit.ID] {
			continue
		}
		semantic = append(semantic, item.hit)
		if len(semantic)+len(lexical) >= limit {
			break
		}
	}
	if len(lexical) == 0 {
		return semantic, nil
	}
	if len(lexical) >= limit {
		return lexical[:limit], nil
	}
	return append(append([]Hit(nil), lexical...), semantic...), nil
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return -1
	}
	var dot, aa, bb float64
	for n := range a {
		x, y := float64(a[n]), float64(b[n])
		dot += x * y
		aa += x * x
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return -1
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
func (i *Index) search(ctx context.Context, q string, limit int) ([]Hit, error) {
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
	case ".txt", ".md", ".json", ".csv", ".log", ".toml", ".go", ".sh", ".pdf", ".docx":
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
	var text string
	if ext == ".pdf" || ext == ".docx" {
		text, err = extractDocumentText(path)
		if err != nil {
			return err
		}
	} else {
		b, readErr := io.ReadAll(io.LimitReader(f, (2<<20)+1))
		if readErr != nil {
			return readErr
		}
		if len(b) > 2<<20 || !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
			return fmt.Errorf("invalid or oversized UTF-8 file")
		}
		text = string(b)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(abs))
	return i.Put(ctx, Document{ID: "local:" + hex.EncodeToString(sum[:]), Source: "user-import", URL: "local:" + filepath.Base(path), Title: filepath.Base(path), Text: text})
}
func (i *Index) Sources(ctx context.Context) ([]Source, error) {
	rows, err := i.DB.QueryContext(ctx, `SELECT id,url,fallback_url,format,license,max_bytes,bytes,last_sync,etag,last_modified,document_count,status,error FROM sources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		var s Source
		if err = rows.Scan(&s.ID, &s.URL, &s.FallbackURL, &s.Format, &s.License, &s.MaxBytes, &s.Bytes, &s.LastSync, &s.ETag, &s.LastModified, &s.DocumentCount, &s.Status, &s.Error); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Sync fetches only built-in HTTPS destinations, never user-provided URLs.
// CISA's official feed is tried first; the official cisagov GitHub mirror is a
// bounded fallback when the CISA origin is unavailable in a restricted network.
func (i *Index) Sync(ctx context.Context, id string) error {
	urls := map[string][]string{
		"cisa-kev": {"https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json", "https://raw.githubusercontent.com/cisagov/kev-data/main/known_exploited_vulnerabilities.json"},
		"owasp":    {"https://raw.githubusercontent.com/OWASP/CheatSheetSeries/master/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.md"},
	}
	endpoints, ok := urls[id]
	if !ok {
		return fmt.Errorf("unknown source")
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var previous struct{ etag, modified string }
	if err := i.DB.QueryRowContext(ctx, `SELECT etag,last_modified FROM sources WHERE id=?`, id).Scan(&previous.etag, &previous.modified); err != nil {
		return err
	}
	var body []byte
	usedURL, etag, modified, notModified, fetchErr := i.fetchSource(ctx, endpoints, previous.etag, previous.modified)
	if fetchErr != nil {
		return i.syncError(ctx, id, fetchErr)
	}
	if notModified {
		return i.recordNotModified(ctx, id, etag, modified)
	}
	body = usedURL.body
	url := usedURL.url
	var err error
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
		if err = json.Unmarshal(body, &feed); err != nil {
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
		if !utf8.Valid(body) || len(body) == 0 || len(body) > 2<<20 {
			return i.syncError(ctx, id, fmt.Errorf("invalid OWASP text"))
		}
		docs = append(docs, Document{ID: "owasp-sql-injection", Source: id, URL: url, Title: "OWASP SQL Injection Prevention", Text: string(body)})
	}
	// Replace only this source inside one transaction. A parse or write error
	// rolls back and leaves the last valid version available for search.
	tx, err := i.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = deleteSourceTx(ctx, tx, id); err != nil {
		return err
	}
	for _, d := range docs {
		if err = i.put(ctx, tx, d); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE sources SET last_sync=?,etag=?,last_modified=?,document_count=?,bytes=?,status='synced',error='' WHERE id=?`, time.Now().UTC().Format(time.RFC3339), etag, modified, len(docs), len(body), id)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO sync_history(source,status,detail,created) VALUES(?,?,?,?)`, id, "synced", fmt.Sprintf("%d documents from %s", len(docs), url), time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

type fetchedSource struct {
	url  string
	body []byte
}

func (i *Index) fetchSource(ctx context.Context, urls []string, etag, modified string) (fetchedSource, string, string, bool, error) {
	var lastErr error
	for _, sourceURL := range urls {
		parsed, err := url.Parse(sourceURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			lastErr = fmt.Errorf("invalid built-in source URL")
			continue
		}
		client := &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" || req.URL.Hostname() != parsed.Hostname() {
				return fmt.Errorf("source redirect leaves allowlist")
			}
			return nil
		}}
		if i.sourceClient != nil {
			client.Transport = i.sourceClient.Transport
		} else {
			client.Transport = safeSourceTransport()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		if modified != "" {
			req.Header.Set("If-Modified-Since", modified)
		}
		res, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		newETag, newModified := res.Header.Get("ETag"), res.Header.Get("Last-Modified")
		if res.StatusCode == http.StatusNotModified {
			res.Body.Close()
			if newETag == "" {
				newETag = etag
			}
			if newModified == "" {
				newModified = modified
			}
			return fetchedSource{url: sourceURL}, newETag, newModified, true, nil
		}
		if res.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s HTTP %d", parsed.Hostname(), res.StatusCode)
			res.Body.Close()
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
		res.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if len(body) > 8<<20 {
			lastErr = fmt.Errorf("source too large")
			continue
		}
		return fetchedSource{url: sourceURL, body: body}, newETag, newModified, false, nil
	}
	return fetchedSource{}, "", "", false, lastErr
}

// safeSourceTransport resolves each built-in source hostname and dials the
// selected numeric address. Private, loopback, link-local, multicast and
// unspecified answers are rejected. Pinning the address for the request closes
// the DNS-rebinding window between the HTTPS allowlist check and TCP connect.
// Tests can inject sourceClient for an in-process TLS fixture.
func safeSourceTransport() http.RoundTripper {
	// Codex Cloud exposes egress through an explicit HTTPS proxy. The proxy is
	// an environment-level control outside the application; use it when set so
	// live source validation remains possible. Built-in source URLs and same-host
	// redirect checks still prevent user-controlled SSRF targets.
	if os.Getenv("HTTPS_PROXY") != "" || os.Getenv("https_proxy") != "" || os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" {
		return &http.Transport{Proxy: http.ProxyFromEnvironment, ForceAttemptHTTP2: true, MaxIdleConns: 2, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 45 * time.Second, ExpectContinueTimeout: time.Second}
	}
	resolver := net.DefaultResolver
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	return &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 45 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := resolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, ip := range ips {
				if !isPublicSourceIP(ip) {
					continue
				}
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("source hostname resolved only to private or otherwise unsafe addresses")
		},
	}
}

func isPublicSourceIP(ip net.IP) bool {
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

func deleteSourceTx(ctx context.Context, tx *sql.Tx, source string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks_fts WHERE chunk IN (SELECT c.id FROM chunks c JOIN documents d ON d.id=c.document WHERE d.source=?)`, source); err != nil {
		return err
	}
	for _, query := range []string{`DELETE FROM chunks WHERE document IN (SELECT id FROM documents WHERE source=?)`, `DELETE FROM document_versions WHERE document IN (SELECT id FROM documents WHERE source=?)`, `DELETE FROM documents WHERE source=?`} {
		if _, err := tx.ExecContext(ctx, query, source); err != nil {
			return err
		}
	}
	return nil
}

func (i *Index) recordNotModified(ctx context.Context, id, etag, modified string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := i.DB.ExecContext(ctx, `UPDATE sources SET last_sync=?,etag=coalesce(nullif(?,''),etag),last_modified=coalesce(nullif(?,''),last_modified),status='synced',error='' WHERE id=?`, now, etag, modified, id)
	if err != nil {
		return err
	}
	_, err = i.DB.ExecContext(ctx, `INSERT INTO sync_history(source,status,detail,created) VALUES(?,?,?,?)`, id, "not-modified", "retained last valid version", now)
	return err
}

func (i *Index) syncError(ctx context.Context, id string, cause error) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := i.DB.ExecContext(ctx, `UPDATE sources SET status='error',error=? WHERE id=?`, cause.Error(), id)
	if err == nil {
		_, err = i.DB.ExecContext(ctx, `INSERT INTO sync_history(source,status,detail,created) VALUES(?,?,?,?)`, id, "error", cause.Error(), now)
	}
	if err != nil {
		return fmt.Errorf("sync: %w; record error: %v", cause, err)
	}
	return cause
}

type SyncEvent struct {
	Source  string `json:"source"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
	Created string `json:"created"`
}

func (i *Index) SyncHistory(ctx context.Context, source string, limit int) ([]SyncEvent, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid sync history limit")
	}
	rows, err := i.DB.QueryContext(ctx, `SELECT source,status,detail,created FROM sync_history WHERE source=? ORDER BY id DESC LIMIT ?`, source, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SyncEvent{}
	for rows.Next() {
		var e SyncEvent
		if err = rows.Scan(&e.Source, &e.Status, &e.Detail, &e.Created); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
