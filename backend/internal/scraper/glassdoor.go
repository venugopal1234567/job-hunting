package scraper

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"remotehunter/internal/models"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// GlassdoorScraper scrapes jobs from Glassdoor using chromedp.
type GlassdoorScraper struct{}

func NewGlassdoorScraper() *GlassdoorScraper {
	return &GlassdoorScraper{}
}

func (s *GlassdoorScraper) Name() string { return "glassdoor" }

func (s *GlassdoorScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.glassdoor.co.in/Job/india-golang-jobs-SRCH_IL.0,5_KO6,12.htm?remoteWorkType=1"
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	} else {
		// Fallback detection logic
		for _, path := range []string{"/usr/bin/chromium-browser", "/usr/bin/google-chrome", "/snap/bin/chromium", "/usr/bin/chromium"} {
			if _, err := os.Stat(path); err == nil {
				opts = append(opts, chromedp.ExecPath(path))
				break
			}
		}
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	// Split by | to support multiple search targets if configured
	urls := strings.Split(targetURL, "|")
	var allJobs []models.Job
	seen := make(map[string]bool)

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		log.Printf("[Scraper] Glassdoor: Navigating to %s", u)
		var jobsJSON string
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
			chromedp.Navigate(u),
			chromedp.Sleep(10*time.Second), // wait for CF challenge & JS execution
			chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 4)`, nil),
			chromedp.Sleep(2*time.Second),
			chromedp.Evaluate(`
(function() {
  const cards = document.querySelectorAll('li[data-test="jobListing"]');
  const results = [];
  cards.forEach(card => {
    const titleEl = card.querySelector('[data-test="job-title"]');
    if (!titleEl) return;
    const title = titleEl.textContent.trim();
    const relativeURL = titleEl.getAttribute('href');
    const jobURL = relativeURL ? new URL(relativeURL, window.location.href).href : '';

    const companyEl = card.querySelector('.EmployerProfile_compactEmployerName__9MGcV') || card.querySelector('[id^="job-employer"] span');
    const company = companyEl ? companyEl.textContent.trim() : '';

    const locEl = card.querySelector('[data-test="emp-location"]');
    const location = locEl ? locEl.textContent.trim() : '';

    const descEl = card.querySelector('[data-test="descSnippet"]');
    const description = descEl ? descEl.textContent.trim() : '';

    results.push({
      title: title,
      company: company,
      url: jobURL,
      location: location,
      description: description
    });
  });
  return JSON.stringify(results);
})()
			`, &jobsJSON),
		)

		if err != nil {
			log.Printf("[Scraper] Glassdoor: chromedp scrape error for %s: %v", u, err)
			continue
		}

		type rawJob struct {
			Title       string `json:"title"`
			Company     string `json:"company"`
			URL         string `json:"url"`
			Location    string `json:"location"`
			Description string `json:"description"`
		}

		var rawJobs []rawJob
		if err := json.Unmarshal([]byte(jobsJSON), &rawJobs); err != nil {
			log.Printf("[Scraper] Glassdoor: failed to unmarshal JSON: %v", err)
			continue
		}

		for _, rj := range rawJobs {
			if rj.Title == "" || rj.Company == "" || rj.URL == "" {
				continue
			}

			country := inferCountryGlassdoor(rj.Location)
			desc := rj.Description
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}
			if desc == "" {
				desc = rj.Title + " at " + rj.Company
			}

			job := &models.Job{
				Title:       rj.Title,
				Company:     rj.Company,
				SourceURL:   rj.URL,
				SourceBoard: "glassdoor",
				Description: desc,
				Location:    rj.Location,
				Country:     country,
				JobType:     "Full Time",
			}

			NormalizeJob(job)
			if !seen[job.JobHash] {
				seen[job.JobHash] = true
				allJobs = append(allJobs, *job)
			}
		}
	}

	log.Printf("[Scraper] Glassdoor: scraped %d unique jobs", len(allJobs))
	return allJobs, nil
}

func inferCountryGlassdoor(location string) string {
	loc := strings.ToLower(location)
	if loc == "" || strings.Contains(loc, "worldwide") || strings.Contains(loc, "anywhere") || loc == "remote" {
		return "Worldwide"
	}

	// Clean punctuation and check for whole word match of "india", "ind", or "in"
	cleanLoc := strings.ReplaceAll(loc, ",", " ")
	cleanLoc = strings.ReplaceAll(cleanLoc, "-", " ")
	words := strings.Fields(cleanLoc)

	isIndia := false
	for _, w := range words {
		if w == "india" || w == "ind" || w == "in" {
			isIndia = true
			break
		}
	}

	if isIndia {
		if strings.Contains(loc, "indiana") || strings.Contains(loc, "indonesia") || strings.Contains(loc, "indies") {
			isIndia = false
		}
	}

	if isIndia {
		return "India"
	}

	cities := []string{"bangalore", "bengaluru", "karnataka", "pune", "hyderabad", "hyderābād", "chennai", "mumbai", "delhi", "noida", "gurgaon", "kolkata", "calcutta", "indore", "gurugram", "cochin", "kochi", "puducherry", "mandamarri"}
	for _, city := range cities {
		if strings.Contains(loc, city) {
			return "India"
		}
	}

	if strings.Contains(loc, "usa") || strings.Contains(loc, "united states") || strings.Contains(loc, "us only") {
		return "US"
	}
	if strings.Contains(loc, "europe") || strings.Contains(loc, "eu only") {
		return "Europe"
	}

	return location
}
