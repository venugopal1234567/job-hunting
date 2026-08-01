package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"remotehunter/internal/models"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// BuiltInScraper uses headless Chromium (via chromedp) to scrape JS-rendered job listings
// from builtin.com. The page is a Drupal SPA that loads jobs client-side.
type BuiltInScraper struct{}

func NewBuiltInScraper() *BuiltInScraper {
	return &BuiltInScraper{}
}

func (s *BuiltInScraper) Name() string { return "builtin" }

func (s *BuiltInScraper) Scrape(targetURL string) ([]models.Job, error) {
	// Build Chrome allocator options optimised for Docker/headless environments
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true), // avoid /dev/shm crash in Docker
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("single-process", false),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
	)

	// Support custom Chrome/Chromium path via env (e.g. CHROME_PATH=/usr/bin/chromium-browser)
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancelCtx()

	// 90-second timeout for the whole scrape
	ctx, cancelTimeout := context.WithTimeout(ctx, 90*time.Second)
	defer cancelTimeout()

	var jobsJSON string

	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),

		// Sleep 6s to allow hydration / JS render
		chromedp.Sleep(6*time.Second),

		// Scroll down to ensure content renders
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 2)`, nil),
		chromedp.Sleep(2*time.Second),

		// Extract all job data via JavaScript
		chromedp.Evaluate(builtInExtractJS, &jobsJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp builtin scrape: %w", err)
	}

	return parseBuiltInJobs(jobsJSON, targetURL)
}

// builtInExtractJS is injected into the page to collect all visible job listings.
// It tries multiple selector strategies because BuiltIn's DOM varies by page type.
const builtInExtractJS = `
(function() {
  const jobs = [];
  const seen = new Set();

  const cards = Array.from(document.querySelectorAll('.job-bounded-responsive, [data-id="job-card"]'));

  cards.forEach(card => {
    // Title is inside a link with data-id="job-card-title"
    const titleEl = card.querySelector('[data-id="job-card-title"], a[href*="/job/"], h2 a, h3 a');
    const title = (titleEl?.innerText || titleEl?.getAttribute('title') || '').trim();

    // Company is inside a link with data-id="company-title" or href*="/company/"
    const companyEl = card.querySelector('[data-id="company-title"], a[href*="/company/"]');
    const company = (companyEl?.innerText || '').trim();

    // Job link
    const linkEl = card.querySelector('a[href*="/job/"]');
    let sourceURL = linkEl?.href || '';
    if (sourceURL && !sourceURL.startsWith('http')) {
      sourceURL = 'https://www.builtin.com' + sourceURL;
    }

    // Description snippet (inside collapse area if available)
    const descEl = card.querySelector('.collapse .text-gray-04, .collapse');
    let desc = (descEl?.innerText || '').trim();
    if (!desc) {
      desc = title + ' at ' + company;
    }

    // Location / Remote
    const locEl = card.querySelector('[data-bs-title], .font-barlow');
    const location = (locEl?.getAttribute('data-bs-title') || locEl?.innerText || 'Remote').replace(/<[^>]*>/g, ' ').trim();

    // Posted date text e.g. "Yesterday", "3 Days Ago"
    const dateEl = card.querySelector('.fa-clock + span, [class*="date"], .text-gray-03');
    const postedAt = dateEl?.innerText || '';

    const key = sourceURL || (title + '|' + company);
    if (title && company && !seen.has(key)) {
      seen.add(key);
      jobs.push({ title, company, location, source_url: sourceURL, salary_range: '', posted_at: postedAt, description: desc });
    }
  });

  return JSON.stringify(jobs);
})()
`

// parseBuiltInJobs unmarshals the JS-extracted JSON into Job models
func parseBuiltInJobs(jobsJSON, fallbackURL string) ([]models.Job, error) {
	var extracted []struct {
		Title       string `json:"title"`
		Company     string `json:"company"`
		Location    string `json:"location"`
		SourceURL   string `json:"source_url"`
		SalaryRange string `json:"salary_range"`
		PostedAt    string `json:"posted_at"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(jobsJSON), &extracted); err != nil {
		return nil, fmt.Errorf("builtin: parse jobs JSON: %w", err)
	}

	var jobs []models.Job
	for _, e := range extracted {
		url := e.SourceURL
		if url == "" {
			url = fallbackURL
		}

		location := e.Location
		if location == "" {
			location = "Remote"
		}

		// Parse posted date if available
		var postedAt *time.Time
		now := time.Now()
		lowerDate := strings.ToLower(e.PostedAt)
		if strings.Contains(lowerDate, "yesterday") || strings.Contains(lowerDate, "1 day") {
			t := now.AddDate(0, 0, -1)
			postedAt = &t
		} else if strings.Contains(lowerDate, "today") || strings.Contains(lowerDate, "hour") {
			postedAt = &now
		} else {
			for _, layout := range []string{time.RFC3339, "2006-01-02", "Jan 2, 2006"} {
				if t, err := time.Parse(layout, strings.TrimSpace(e.PostedAt)); err == nil {
					postedAt = &t
					break
				}
			}
		}

		desc := e.Description
		if desc == "" {
			desc = fmt.Sprintf("%s at %s — %s", e.Title, e.Company, location)
		}

		job := &models.Job{
			Title:       e.Title,
			Company:     e.Company,
			Location:    location,
			Country:     inferCountry(location),
			SourceURL:   url,
			SourceBoard: "builtin",
			SalaryRange: e.SalaryRange,
			Description: desc,
			PostedAt:    postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] BuiltIn (chromedp): scraped %d jobs", len(jobs))
	return jobs, nil
}
