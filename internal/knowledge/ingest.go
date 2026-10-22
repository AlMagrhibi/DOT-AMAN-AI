package knowledge

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

const maxExtractedDocumentBytes = 16 << 20

// extractDocumentText reads bounded local documents. It never executes macros,
// relationships, embedded files or document instructions.
func extractDocumentText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return extractPDF(path)
	case ".docx":
		return extractDOCX(path)
	default:
		return "", fmt.Errorf("unsupported document format: %s", ext)
	}
}

func extractPDF(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxExtractedDocumentBytes {
		return "", fmt.Errorf("PDF must be a regular file no larger than %d bytes", maxExtractedDocumentBytes)
	}
	f, reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open PDF: %w", err)
	}
	defer f.Close()
	plain, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract PDF text: %w", err)
	}
	b, err := io.ReadAll(io.LimitReader(plain, maxExtractedDocumentBytes+1))
	if err != nil {
		return "", err
	}
	if len(b) > maxExtractedDocumentBytes {
		return "", fmt.Errorf("extracted PDF text exceeds limit")
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return "", fmt.Errorf("PDF has no extractable text")
	}
	return text, nil
}

func extractDOCX(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxExtractedDocumentBytes {
		return "", fmt.Errorf("DOCX must be a regular file no larger than %d bytes", maxExtractedDocumentBytes)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open DOCX: %w", err)
	}
	defer z.Close()
	var document *zip.File
	var expanded uint64
	for _, file := range z.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > maxExtractedDocumentBytes || expanded+file.UncompressedSize64 > maxExtractedDocumentBytes {
			return "", fmt.Errorf("DOCX expanded content exceeds limit")
		}
		expanded += file.UncompressedSize64
		if file.Name == "word/document.xml" {
			document = file
		}
	}
	if document == nil {
		return "", fmt.Errorf("DOCX document.xml missing")
	}
	r, err := document.Open()
	if err != nil {
		return "", err
	}
	defer r.Close()
	decoder := xml.NewDecoder(io.LimitReader(r, maxExtractedDocumentBytes+1))
	var b strings.Builder
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return "", fmt.Errorf("parse DOCX XML: %w", tokenErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "t":
				var text string
				if err = decoder.DecodeElement(&text, &value); err != nil {
					return "", err
				}
				b.WriteString(text)
			case "tab":
				b.WriteByte('\t')
			case "br", "cr", "p":
				b.WriteByte('\n')
			}
		}
		if b.Len() > maxExtractedDocumentBytes {
			return "", fmt.Errorf("extracted DOCX text exceeds limit")
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("DOCX has no extractable text")
	}
	return text, nil
}
