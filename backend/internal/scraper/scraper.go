package scraper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"remotehunter/internal/models"
	"time"

	"github.com/chromedp/chromedp"
)

// Scraper is the interface all job board scrapers must implement
type Scraper interface {
	Name() string
	Scrape(targetURL string) ([]models.Job, error)
}

// chromeUserAgent is the desktop UA sent by all headless-browser scrapers.
const chromeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// chromeFallbackPaths are probed, in order, when CHROME_PATH is unset.
var chromeFallbackPaths = []string{"/usr/bin/chromium-browser", "/usr/bin/google-chrome", "/snap/bin/chromium", "/usr/bin/chromium"}

// newHeadlessContext returns a chromedp context for scraping JS-rendered job
// boards: headless Chromium with anti-automation flags, an explicit binary
// path when one can be found, and a hard timeout. The cancel func must be
// deferred; it releases the timeout, browser context and allocator in that
// order so no browser process is leaked.
func newHeadlessContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent(chromeUserAgent),
	)

	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	} else {
		for _, path := range chromeFallbackPaths {
			if _, err := os.Stat(path); err == nil {
				opts = append(opts, chromedp.ExecPath(path))
				break
			}
		}
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, timeout)

	return timeoutCtx, func() {
		cancelTimeout()
		cancelCtx()
		cancelAlloc()
	}
}

// ComputeJobHash creates a SHA256 deduplication hash for a job
func ComputeJobHash(title, company, sourceURL string) string {
	h := sha256.New()
	h.Write([]byte(title + "|" + company + "|" + sourceURL))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NormalizeJob sets defaults and computes the hash for a scraped job
func NormalizeJob(job *models.Job) {
	job.ScrapedAt = time.Now()
	job.IsActive = true
	job.JobHash = ComputeJobHash(job.Title, job.Company, job.SourceURL)
}

// isGoRole reports whether a posting is a Go role, judged on the title. It
// reuses the word-boundary regex from remoteok.go, which excludes non-Go titles
// that merely contain the letters (e.g. "Going" or a "go-to-market" manager).
// For boards whose API returns unfiltered results.
func isGoRole(title string) bool {
	return goWordRegex.MatchString(title) && !excludeTitleRegex.MatchString(title)
}
