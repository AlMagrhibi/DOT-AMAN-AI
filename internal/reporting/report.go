package reporting

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Finding is evidence-first. Empty scoring fields mean that the source did
// not provide a verified CVSS/EPSS value; the exporter never invents scores.
type Finding struct {
	ID             string   `json:"id,omitempty"`
	Title          string   `json:"title"`
	Severity       string   `json:"severity,omitempty"`
	CVE            string   `json:"cve,omitempty"`
	CWE            string   `json:"cwe,omitempty"`
	CVSS           string   `json:"cvss,omitempty"`
	EPSS           string   `json:"epss,omitempty"`
	KEV            *bool    `json:"kev,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	Verification   string   `json:"verification,omitempty"`
}

type Evidence struct {
	Tool     string `json:"tool"`
	Status   string `json:"status"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration int64  `json:"duration_ns"`
}

type Reference struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
}

// SecurityReport is a stable Arabic-first interchange format. It separates
// observed evidence from recommendations and records unavailable fields.
type SecurityReport struct {
	SchemaVersion   string      `json:"schema_version"`
	Language        string      `json:"language"`
	Session         string      `json:"session"`
	GeneratedAt     string      `json:"generated_at"`
	Scope           string      `json:"authorized_scope"`
	Summary         string      `json:"summary"`
	Evidence        []Evidence  `json:"evidence"`
	Findings        []Finding   `json:"findings"`
	References      []Reference `json:"references"`
	Recommendations []string    `json:"recommendations"`
	Limitations     []string    `json:"limitations"`
}

func Build(o app.Outcome) SecurityReport {
	r := SecurityReport{
		SchemaVersion: "1.0",
		Language:      "ar",
		Session:       o.Session,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Scope:         "local system and explicitly supplied files only; authorization must be confirmed by the operator",
		Summary:       o.Text,
		Recommendations: []string{
			"تحقق من كل نتيجة على النظام المصرح به قبل تطبيق أي إصلاح.",
			"لا تعتبر النصوص المسترجعة أو ملخصات النموذج دليلًا مستقلًا دون تحقق.",
		},
		Limitations: []string{
			"الدرجات CVSS/EPSS وحالة CISA KEV غير متاحة ما لم يوردها مصدر موثق.",
			"الاستنتاجات لا تثبت تأثر النظام دون نتيجة أداة فعلية.",
		},
	}
	for _, x := range o.Results {
		status := "complete"
		if x.Error != "" {
			status = "failed"
		}
		r.Evidence = append(r.Evidence, Evidence{Tool: x.Name, Status: status, Output: x.Output, Error: x.Error, Duration: x.Duration.Nanoseconds()})
	}
	seen := map[string]bool{}
	for _, h := range o.Citations {
		if seen[h.ID] {
			continue
		}
		seen[h.ID] = true
		r.References = append(r.References, Reference{ID: h.ID, Title: h.Title, URL: h.URL, SHA256: h.SHA256})
	}
	if len(r.Evidence) == 0 {
		r.Evidence = []Evidence{}
	}
	if len(r.Findings) == 0 {
		r.Findings = []Finding{}
	}
	if len(r.References) == 0 {
		r.References = []Reference{}
	}
	return r
}

func markdown(r SecurityReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# تقرير Dot Aman AI الأمني\n\n- الجلسة: `%s`\n- وقت الإنشاء: `%s`\n- النطاق المصرح: %s\n\n## الملخص\n\n%s\n\n## الأدلة الفعلية\n", r.Session, r.GeneratedAt, r.Scope, r.Summary)
	if len(r.Evidence) == 0 {
		b.WriteString("\nلا توجد أدوات منفذة في هذه النتيجة.\n")
	}
	for _, e := range r.Evidence {
		fmt.Fprintf(&b, "\n### %s — %s\n\nالمدة: %dns\n\n```text\n%s\n```\n", e.Tool, e.Status, e.Duration, e.Output)
		if e.Error != "" {
			fmt.Fprintf(&b, "الخطأ: `%s`\n", e.Error)
		}
	}
	b.WriteString("\n## النتائج\n\n")
	if len(r.Findings) == 0 {
		b.WriteString("لم تُسجل ثغرة مؤكدة في هذه النتيجة. لا يعني ذلك أن الفحص شامل.\n")
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "- **%s** (%s)\n  - CVE: %s · CWE: %s · CVSS: %s · EPSS: %s · KEV: %v\n  - الدليل: %s\n  - التوصية: %s\n  - التحقق: %s\n", f.Title, f.Severity, valueOrUnavailable(f.CVE), valueOrUnavailable(f.CWE), valueOrUnavailable(f.CVSS), valueOrUnavailable(f.EPSS), f.KEV, strings.Join(f.Evidence, "؛ "), f.Recommendation, f.Verification)
	}
	b.WriteString("\n## المراجع\n\n")
	for _, ref := range r.References {
		fmt.Fprintf(&b, "- [%s](%s) — %s\n", ref.ID, ref.URL, ref.Title)
	}
	b.WriteString("\n## التوصيات والقيود\n\n")
	for _, x := range r.Recommendations {
		fmt.Fprintf(&b, "- %s\n", x)
	}
	for _, x := range r.Limitations {
		fmt.Fprintf(&b, "- قيد: %s\n", x)
	}
	return b.String()
}

func valueOrUnavailable(s string) string {
	if s == "" {
		return "غير متاح"
	}
	return s
}

func Export(dir string, o app.Outcome) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	prefix := filepath.Join(dir, "report-"+time.Now().UTC().Format("20060102T150405.000000000"))
	legacy, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return "", err
	}
	structuredReport := Build(o)
	structured, err := json.MarshalIndent(structuredReport, "", "  ")
	if err != nil {
		return "", err
	}
	files := map[string][]byte{".json": legacy, ".security.json": structured, ".md": []byte(markdown(structuredReport))}
	for ext, text := range files {
		f, err := os.OpenFile(prefix+ext, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return "", err
		}
		_, err = f.Write(text)
		closeErr := f.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	// PDF is an optional local renderer because deployment images may not carry
	// a Unicode font. Markdown/JSON are already complete and remain available.
	if err := ExportPDF(prefix+".pdf", structuredReport); err != nil && !errors.Is(err, ErrPDFUnavailable) {
		return "", err
	}
	return prefix, nil
}
