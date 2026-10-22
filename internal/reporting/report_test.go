package reporting

import (
	"encoding/json"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/tools"
	"os"
	"strings"
	"testing"
)

func TestReportEvidenceAndPermissions(t *testing.T) {
	o := app.Outcome{Session: "synthetic", Text: "نتيجة موثقة", Results: []tools.Result{{Name: "system_info", Output: "Linux"}}}
	p, err := Export(t.TempDir(), o)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var got app.Outcome
	if err = json.Unmarshal(b, &got); err != nil || got.Results[0].Output != "Linux" {
		t.Fatal(got, err)
	}
	structured, err := os.ReadFile(p + ".security.json")
	if err != nil {
		t.Fatal(err)
	}
	var report SecurityReport
	if err = json.Unmarshal(structured, &report); err != nil || report.SchemaVersion != "1.0" || report.Language != "ar" || len(report.Evidence) != 1 || len(report.Limitations) == 0 {
		t.Fatalf("structured report: %+v err=%v", report, err)
	}
	md, err := os.ReadFile(p + ".md")
	if err != nil || !strings.Contains(string(md), "CVSS") || !strings.Contains(string(md), "الأدلة") {
		t.Fatalf("markdown report missing required fields: %v", err)
	}
	info, err := os.Stat(p + ".json")
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, err)
	}
}
