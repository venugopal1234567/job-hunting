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

type BaytScraper struct{}

func NewBaytScraper() *BaytScraper {
	return &BaytScraper{}
}

func (s *BaytScraper) Name() string { return "bayt" }

func (s *BaytScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.bayt.com/en/international/jobs/golang-remote-jobs/"
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

	urls := strings.Split(targetURL, "|")
	var allJobs []models.Job
	seen := make(map[string]bool)

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		log.Printf("[Scraper] Bayt: Navigating to %s", u)
		var jobsJSON string
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
			chromedp.Navigate(u),
			chromedp.Sleep(8*time.Second),
			chromedp.Evaluate(`
(function() {
  const cards = document.querySelectorAll('li[data-js-job]');
  const results = [];
  cards.forEach(card => {
    const titleEl = card.querySelector('h2 a');
    if (!titleEl) return;
    const title = titleEl.textContent.trim();
    const href = titleEl.getAttribute('href');
    const jobURL = href ? new URL(href, window.location.href).href : '';

    const companyEl = card.querySelector('.job-company-location-wrapper a.t-bold') || card.querySelector('a.t-default.t-bold');
    const company = companyEl ? companyEl.textContent.trim() : 'N/A';

    const locEl = card.querySelector('.job-company-location-wrapper div.t-mute span') || card.querySelector('a.t-mute span');
    const location = locEl ? locEl.textContent.trim() : 'Remote';

    const descEl = card.querySelector('div.jb-descr');
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
			log.Printf("[Scraper] Bayt: chromedp scrape error for %s: %v", u, err)
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
			log.Printf("[Scraper] Bayt: failed to parse JSON: %v", err)
			continue
		}

		for _, rj := range rawJobs {
			if rj.Title == "" || rj.URL == "" {
				continue
			}

			country := inferCountryBayt(rj.Location)
			desc := rj.Description
			if desc == "" {
				desc = rj.Title + " at " + rj.Company
			}
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}

			job := &models.Job{
				Title:       rj.Title,
				Company:     rj.Company,
				SourceURL:   rj.URL,
				SourceBoard: "bayt",
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

	log.Printf("[Scraper] Bayt: scraped %d unique jobs", len(allJobs))
	return allJobs, nil
}

func inferCountryBayt(location string) string {
	loc := strings.ToLower(location)
	if loc == "" || strings.Contains(loc, "worldwide") || strings.Contains(loc, "anywhere") || loc == "remote" {
		return "Worldwide"
	}

	cities := []string{"bangalore", "bengaluru", "karnataka", "pune", "hyderabad", "hyderābād", "chennai", "mumbai", "delhi", "noida", "gurgaon", "kolkata", "calcutta", "indore", "gurugram", "cochin", "kochi", "puducherry", "mandamarri"}
	for _, city := range cities {
		if strings.Contains(loc, city) {
			return "India"
		}
	}

	if strings.Contains(loc, "india") || strings.Contains(loc, " ind ") || strings.Contains(loc, " in ") {
		return "India"
	}

	return "Worldwide"
}
