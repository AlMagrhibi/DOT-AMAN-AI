package reporting

import (
	"encoding/json"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"os"
	"path/filepath"
	"time"
)

func Export(dir string, o app.Outcome) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	prefix := filepath.Join(dir, "report-"+time.Now().UTC().Format("20060102T150405.000000000"))
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return "", err
	}
	md := fmt.Sprintf("# تقرير Dot Aman AI\n\nالجلسة: `%s`\n\n%s\n\n## دليل التنفيذ الفعلي\n", o.Session, o.Text)
	for _, r := range o.Results {
		md += fmt.Sprintf("\n### %s\n\nالخطأ: %s\n\n```text\n%s\n```\n", r.Name, r.Error, r.Output)
	}
	for ext, text := range map[string][]byte{".json": b, ".md": []byte(md)} {
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
	return prefix, nil
}
