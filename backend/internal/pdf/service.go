// Package pdf provides a service for rendering HTML content into PDF
// via a headless Chromium process with proper timeout and cleanup guarantees.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DefaultTimeout is the maximum time allowed for Chromium to complete PDF
// generation. If the process does not exit within this window it is killed and
// an error is returned so the request never hangs indefinitely.
const DefaultTimeout = 60 * time.Second

// Service handles all PDF generation concerns.
type Service struct {
	chromiumBin string
	tempDir     string
}

// New creates a Service, resolving the Chromium binary path and the directory
// used for temporary files. It returns an error if no Chromium binary can be
// found on the host.
func New(tempDir string) (*Service, error) {
	bin := findChromiumBinary()
	if bin == "" {
		return nil, fmt.Errorf("no chromium binary found; install chromium or google-chrome")
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	return &Service{chromiumBin: bin, tempDir: tempDir}, nil
}

// GenerateFromHTML converts htmlContent into a PDF and returns the raw bytes.
//
// Two temporary files are created (HTML + PDF) and both are removed via defer
// statements that execute even if the function panics – they are registered
// before any fallible operation writes to them.
func (s *Service) GenerateFromHTML(ctx context.Context, htmlContent string) ([]byte, error) {
	// ── Create HTML temp file ─────────────────────────────────────────────
	htmlFile, err := os.CreateTemp(s.tempDir, "resume_*.html")
	if err != nil {
		return nil, fmt.Errorf("create temp html: %w", err)
	}
	htmlPath := htmlFile.Name()
	// Register cleanup immediately so a later panic still removes the file.
	defer func() {
		if rerr := os.Remove(htmlPath); rerr != nil && !os.IsNotExist(rerr) {
			log.Printf("[pdf] failed to remove temp html %s: %v", htmlPath, rerr)
		}
	}()

	if _, err = htmlFile.WriteString(htmlContent); err != nil {
		_ = htmlFile.Close()
		return nil, fmt.Errorf("write temp html: %w", err)
	}
	if err = htmlFile.Close(); err != nil {
		return nil, fmt.Errorf("close temp html: %w", err)
	}

	// ── Create PDF temp file ──────────────────────────────────────────────
	pdfFile, err := os.CreateTemp(s.tempDir, "resume_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp pdf: %w", err)
	}
	pdfPath := pdfFile.Name()
	// Register cleanup immediately – same rationale as above.
	defer func() {
		if rerr := os.Remove(pdfPath); rerr != nil && !os.IsNotExist(rerr) {
			log.Printf("[pdf] failed to remove temp pdf %s: %v", pdfPath, rerr)
		}
	}()
	// We only needed the path; close the file so Chromium can write to it.
	if err = pdfFile.Close(); err != nil {
		return nil, fmt.Errorf("close temp pdf: %w", err)
	}

	// ── Run Chromium with timeout ─────────────────────────────────────────
	if err = s.runChromium(ctx, htmlPath, pdfPath); err != nil {
		return nil, err
	}

	pdfBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read generated pdf: %w", err)
	}
	if len(pdfBytes) == 0 {
		return nil, fmt.Errorf("chromium produced an empty PDF")
	}
	return pdfBytes, nil
}

// runChromium executes the headless Chromium print-to-PDF command inside a
// context-scoped timeout so that zombie processes cannot accumulate.
func (s *Service) runChromium(ctx context.Context, htmlPath, pdfPath string) error {
	// Apply a hard timeout if the caller did not already constrain the context.
	timeoutCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, s.chromiumBin,
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--no-pdf-header-footer",
		"--print-to-pdf="+pdfPath,
		htmlPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// context.DeadlineExceeded means Chromium was killed due to the timeout.
		if timeoutCtx.Err() != nil {
			return fmt.Errorf("chromium timed out after %s", DefaultTimeout)
		}
		return fmt.Errorf("chromium: %w; stderr: %s", err, stderr.String())
	}
	return nil
}

// findChromiumBinary returns the first usable Chromium executable path, or an
// empty string when none is installed.
func findChromiumBinary() string {
	knownPaths := []string{
		"/snap/bin/chromium",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
	}
	for _, p := range knownPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fall back to PATH resolution.
	for _, name := range []string{"chromium", "google-chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// DefaultTempDir returns the OS-appropriate temp directory for this service.
func DefaultTempDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "tmp", "remotehunter")
}

// InjectSinglePageStyle inserts a compact CSS override block into htmlContent
// to force all content onto one printed page. It targets </head> when present,
// otherwise prepends the style block.
func InjectSinglePageStyle(htmlContent string) string {
	style := `<style id="fit-single-page-override">
@page { size: letter; margin: 5mm 8mm !important; }
* { box-sizing: border-box !important; }
html, body { font-size: 8.5pt !important; line-height: 1.15 !important; padding: 0 !important; margin: 0 auto !important; width: 100% !important; max-width: 100% !important; height: 100% !important; overflow: hidden !important; }
header { margin-bottom: 2px !important; }
h1 { font-size: 14pt !important; margin: 0 0 1px 0 !important; letter-spacing: 0.5px !important; }
.subtitle { font-size: 9.5pt !important; margin: 0 0 2px 0 !important; }
.contact-info { font-size: 8pt !important; margin-top: 1px !important; gap: 6px !important; }
.contact-info span { font-size: 8pt !important; }
.contact-info svg { width: 8px !important; height: 8px !important; margin-right: 2px !important; }
h2 { font-size: 9.5pt !important; margin-top: 3px !important; margin-bottom: 2px !important; padding-bottom: 1px !important; border-bottom: 1px solid #000 !important; }
p { margin: 0 0 2px 0 !important; font-size: 8.5pt !important; line-height: 1.15 !important; }
.job-title-container { margin-bottom: 0px !important; font-size: 9pt !important; }
.job-title { font-size: 9pt !important; }
.job-date { font-size: 8.5pt !important; }
.company-container { font-size: 8.5pt !important; margin-bottom: 1px !important; }
ul { margin: 0 0 2px 0 !important; padding-left: 14px !important; font-size: 8.5pt !important; }
li { margin-bottom: 1px !important; line-height: 1.15 !important; font-size: 8.5pt !important; }
.tech-used { font-size: 8pt !important; margin-top: 1px !important; margin-bottom: 3px !important; }
.edu-details { font-size: 8.5pt !important; margin-top: 0px !important; margin-bottom: 2px !important; }
.skills-table { font-size: 8.5pt !important; margin-bottom: 2px !important; width: 100% !important; }
.skills-table td { padding: 0.5px 0 !important; font-size: 8.5pt !important; }
section, .job-title-container, .company-container, ul, table { page-break-inside: avoid !important; break-inside: avoid !important; }
</style>`

	if strings.Contains(htmlContent, "</head>") {
		return strings.Replace(htmlContent, "</head>", style+"</head>", 1)
	}
	return style + htmlContent
}
