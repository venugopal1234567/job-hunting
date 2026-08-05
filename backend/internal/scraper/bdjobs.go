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

type BDJobsScraper struct{}

func NewBDJobsScraper() *BDJobsScraper {
	return &BDJobsScraper{}
}

func (s *BDJobsScraper) Name() string { return "bdjobs" }

func (s *BDJobsScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://jobs.bdjobs.com/jobsearch.asp?txtsearch=golang"
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

		log.Printf("[Scraper] BDJobs: Navigating to %s", u)
		var jobsJSON string
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
			chromedp.Navigate(u),
			chromedp.Sleep(8*time.Second),
			chromedp.Evaluate(`
(function() {
  const cards = [];
  // Find all potential job elements
  document.querySelectorAll('div.job-item, div.sout-jobs-wrapper, div.norm-jobs-wrapper, div.featured-wrap').forEach(el => {
    cards.push(el);
  });
  
  // If not found, try fallback of getting parent elements of jobdetail links
  if (cards.length === 0) {
    document.querySelectorAll('a').forEach(a => {
      if (a.getAttribute('href') && a.getAttribute('href').toLowerCase().includes('jobdetail')) {
        let parent = a.parentElement;
        // search up to 4 parents for a wrapper container
        for (let i = 0; i < 4; i++) {
          if (parent && (parent.classList.contains('job-item') || parent.classList.contains('sout-jobs-wrapper') || parent.tagName === 'DIV')) {
            if (!cards.includes(parent)) {
              cards.push(parent);
            }
            break;
          }
          if (parent) parent = parent.parentElement;
        }
      }
    });
  }

  const results = [];
  cards.forEach(card => {
    const jobLink = card.querySelector('a[href*="jobdetail"]') || card.querySelector('a[href*="JobDetail"]');
    if (!jobLink) return;

    const title = jobLink.textContent.trim();
    const href = jobLink.getAttribute('href');
    const jobURL = href ? new URL(href, window.location.href).href : '';

    const companyEl = card.querySelector('.comp-name-text') || card.querySelector('.company-name') || card.querySelector('[class*="comp-name"]');
    const company = companyEl ? companyEl.textContent.trim() : 'N/A';

    const locEl = card.querySelector('.locon-text-d') || card.querySelector('[class*="locon"]');
    const location = locEl ? locEl.textContent.trim() : 'Bangladesh';

    const dateEl = card.querySelector('[class*="date"]') || card.querySelector('[class*="deadline"]') || card.querySelector('[class*="published"]');
    const dateText = dateEl ? dateEl.textContent.trim() : '';

    results.push({
      title: title,
      company: company,
      url: jobURL,
      location: location,
      dateText: dateText
    });
  });
  return JSON.stringify(results);
})()
			`, &jobsJSON),
		)

		if err != nil {
			log.Printf("[Scraper] BDJobs: chromedp scrape error for %s: %v", u, err)
			continue
		}

		type rawJob struct {
			Title    string `json:"title"`
			Company  string `json:"company"`
			URL      string `json:"url"`
			Location string `json:"location"`
			DateText string `json:"dateText"`
		}

		var rawJobs []rawJob
		if err := json.Unmarshal([]byte(jobsJSON), &rawJobs); err != nil {
			log.Printf("[Scraper] BDJobs: failed to parse JSON: %v", err)
			continue
		}

		for _, rj := range rawJobs {
			if rj.Title == "" || rj.URL == "" {
				continue
			}

			// Clean up location
			location := rj.Location
			if location == "" {
				location = "Bangladesh"
			}

			// Determine country (BDJobs is primarily Bangladesh)
			country := "Bangladesh"
			if strings.Contains(strings.ToLower(location), "remote") || strings.Contains(strings.ToLower(rj.Title), "remote") {
				country = "Worldwide"
			}

			desc := rj.Title + " at " + rj.Company + " (Location: " + location + ")"
			if rj.DateText != "" {
				desc += "\nDeadline: " + rj.DateText
			}

			job := &models.Job{
				Title:       rj.Title,
				Company:     rj.Company,
				SourceURL:   rj.URL,
				SourceBoard: "bdjobs",
				Description: desc,
				Location:    location,
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

	log.Printf("[Scraper] BDJobs: scraped %d unique jobs", len(allJobs))
	return allJobs, nil
}
