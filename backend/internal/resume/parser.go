package resume

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"unicode"
)

// ParseText reads plain text or basic PDF text extraction from a reader
// For PDF files, we do simple text extraction by stripping binary content
func ParseText(r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	// Check if it's a PDF (starts with %PDF)
	raw := string(content)
	if strings.HasPrefix(raw, "%PDF") {
		return extractTextFromPDF(content), nil
	}

	// Plain text - just clean it up
	return cleanText(raw), nil
}

// extractTextFromPDF uses pdftotext to extract clean text from raw PDF bytes
func extractTextFromPDF(data []byte) string {
	// Try pdftotext CLI first
	cmd := exec.Command("pdftotext", "-", "-")
	cmd.Stdin = bytes.NewReader(data)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil && out.Len() > 0 {
		return cleanText(out.String())
	}

	// Fallback to basic string extraction
	var sb strings.Builder
	var currentWord strings.Builder

	for _, b := range data {
		r := rune(b)
		if unicode.IsPrint(r) && r != '\\' {
			currentWord.WriteRune(r)
		} else {
			word := strings.TrimSpace(currentWord.String())
			if len(word) > 2 {
				sb.WriteString(word)
				sb.WriteRune(' ')
			}
			currentWord.Reset()
		}
	}

	return cleanText(sb.String())
}

// cleanText normalizes excess spaces but keeps original formatting structure
func cleanText(s string) string {
	// Replaces carriage returns with standard newlines and returns trimmed string.
	// We do NOT flatten newlines into a single space here so that resume document layout stays correct.
	return strings.TrimSpace(strings.ReplaceAll(s, "\r", ""))
}
