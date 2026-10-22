package reporting

import (
	"errors"
	"fmt"
	"github.com/phpdave11/gofpdf"
	"os"
	"path/filepath"
)

var ErrPDFUnavailable = errors.New("PDF export requires a local Unicode TrueType font")

func pdfFontPath() string {
	for _, path := range []string{
		os.Getenv("DOTAMAN_PDF_FONT"),
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
	} {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

// ExportPDF writes a local Unicode PDF when the host supplies a TrueType font.
// It is intentionally explicit: a missing font is reported as unavailable,
// never silently replaced with a non-Arabic built-in PDF font.
func ExportPDF(path string, report SecurityReport) error {
	font := pdfFontPath()
	if font == "" {
		return ErrPDFUnavailable
	}
	pdf := gofpdf.New("P", "mm", "A4", filepath.Dir(font))
	pdf.SetTitle("Dot Aman AI Security Report", true)
	pdf.AddUTF8Font("dejavu", "", filepath.Base(font))
	if pdf.Error() != nil {
		return fmt.Errorf("load PDF font: %w", pdf.Error())
	}
	pdf.AddPage()
	pdf.SetFont("dejavu", "", 12)
	pdf.MultiCell(0, 7, "تقرير Dot Aman AI الأمني", "", "R", false)
	pdf.SetFont("dejavu", "", 9)
	pdf.MultiCell(0, 5, "الجلسة: "+report.Session+"\nوقت الإنشاء: "+report.GeneratedAt+"\nالنطاق: "+report.Scope+"\n\nالملخص:\n"+report.Summary, "", "R", false)
	pdf.Ln(3)
	pdf.SetFont("dejavu", "", 10)
	pdf.MultiCell(0, 6, "الأدلة الفعلية", "", "R", false)
	for _, evidence := range report.Evidence {
		text := evidence.Tool + " — " + evidence.Status + "\n" + evidence.Output
		if evidence.Error != "" {
			text += "\nالخطأ: " + evidence.Error
		}
		pdf.MultiCell(0, 5, text, "", "R", false)
		pdf.Ln(1)
	}
	pdf.MultiCell(0, 6, "التوصيات والقيود", "", "R", false)
	for _, item := range append(append([]string{}, report.Recommendations...), report.Limitations...) {
		pdf.MultiCell(0, 5, "- "+item, "", "R", false)
	}
	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}
	return nil
}
