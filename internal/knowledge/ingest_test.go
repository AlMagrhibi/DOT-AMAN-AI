package knowledge

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportDOCXAndPDFText(t *testing.T) {
	dir := t.TempDir()
	docx := filepath.Join(dir, "security.docx")
	file, err := os.Create(docx)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	entry, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="urn:schemas-microsoft-com:office:word"><w:body><w:p><w:r><w:t>Least privilege</w:t></w:r></w:p><w:p><w:r><w:t>صلاحيات آمنة</w:t></w:r></w:p></w:body></w:document>`))
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	text, err := extractDocumentText(docx)
	if err != nil || !strings.Contains(text, "Least privilege") || !strings.Contains(text, "صلاحيات") {
		t.Fatalf("DOCX text=%q err=%v", text, err)
	}

	pdfPath := filepath.Join(dir, "security.pdf")
	if err = os.WriteFile(pdfPath, minimalPDF("Least privilege"), 0600); err != nil {
		t.Fatal(err)
	}
	text, err = extractDocumentText(pdfPath)
	if err != nil || !strings.Contains(text, "Least privilege") {
		t.Fatalf("PDF text=%q err=%v", text, err)
	}
}

func TestImportRejectsUnsupportedAndOversizedArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.docx")
	if err := os.WriteFile(path, []byte("not zip"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := extractDocumentText(path); err == nil {
		t.Fatal("invalid DOCX accepted")
	}
}

func minimalPDF(text string) []byte {
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 300] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n",
		fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\nBT /F1 12 Tf 72 200 Td (%s) Tj ET\nendstream\nendobj\n", len(text)+31, text),
		"5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for _, object := range objects {
		offsets = append(offsets, b.Len())
		b.WriteString(object)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return b.Bytes()
}
