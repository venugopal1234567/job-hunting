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

type NaukriScraper struct{}

func NewNaukriScraper() *NaukriScraper {
	return &NaukriScraper{}
}

func (s *NaukriScraper) Name() string { return "naukri" }

func (s *NaukriScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.naukri.com/golang-jobs?wfhType=2"
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

		log.Printf("[Scraper] Naukri: Navigating to %s", u)
		var jobsJSON string
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
			chromedp.Navigate(u),
			chromedp.Sleep(10*time.Second),
			chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 4)`, nil),
			chromedp.Sleep(2*time.Second),
			chromedp.Evaluate(`
(function() {
  const cards = [];
  // Find cards using known class names
  document.querySelectorAll('.srp-jobtuple-container, .cust-job-tuple, [class*="jobtuple"], [class*="job-tuple"]').forEach(el => {
    cards.push(el);
  });
  
  // Fallback: search for anchor tags linking to job listings
  if (cards.length === 0) {
    document.querySelectorAll('a').forEach(a => {
      const href = a.getAttribute('href') || '';
      if (href.includes('job-listings-') || href.includes('/job-listings')) {
        let parent = a.parentElement;
        for (let i = 0; i < 5; i++) {
          if (parent && (parent.classList.contains('srp-jobtuple-container') || parent.classList.contains('cust-job-tuple') || parent.tagName === 'DIV')) {
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
    const titleEl = card.querySelector('a.title') || card.querySelector('a[href*="job-listings-"]') || card.querySelector('a[href*="/job-listings"]');
    if (!titleEl) return;
    const title = titleEl.textContent.trim();
    const jobURL = titleEl.getAttribute('href') || '';

    const companyEl = card.querySelector('a.comp-name') || card.querySelector('.comp-name') || card.querySelector('.companyName') || card.querySelector('[class*="comp"]');
    const company = companyEl ? companyEl.textContent.trim() : 'N/A';

    const locEl = card.querySelector('.locWdth') || card.querySelector('.location') || card.querySelector('[class*="loc"]');
    const location = locEl ? locEl.textContent.trim() : 'Remote';

    const descEl = card.querySelector('.job-desc') || card.querySelector('[class*="desc"]') || card.querySelector('.jobDescription');
    const description = descEl ? descEl.textContent.trim() : '';

    const dateEl = card.querySelector('.job-post-day') || card.querySelector('[class*="post-day"]') || card.querySelector('[class*="date"]');
    const dateText = dateEl ? dateEl.textContent.trim() : '';

    results.push({
      title: title,
      company: company,
      url: jobURL,
      location: location,
      description: description,
      dateText: dateText
    });
  });
  return JSON.stringify(results);
})()
			`, &jobsJSON),
		)

		if err != nil {
			log.Printf("[Scraper] Naukri: chromedp scrape error for %s: %v", u, err)
			continue
		}

		if jobsJSON == "" || jobsJSON == "[]" {
			var pageTitle, bodySnippet string
			_ = chromedp.Run(ctx,
				chromedp.Evaluate(`document.title`, &pageTitle),
				chromedp.Evaluate(`document.body ? document.body.innerText.substring(0, 500) : "No body"`, &bodySnippet),
			)
			log.Printf("[Scraper] Naukri DEBUG: No jobs found. Page Title: '%s', Content: '%s'", pageTitle, bodySnippet)
		}

		type rawJob struct {
			Title       string `json:"title"`
			Company     string `json:"company"`
			URL         string `json:"url"`
			Location    string `json:"location"`
			Description string `json:"description"`
			DateText    string `json:"dateText"`
		}

		var rawJobs []rawJob
		if err := json.Unmarshal([]byte(jobsJSON), &rawJobs); err != nil {
			log.Printf("[Scraper] Naukri: failed to parse JSON: %v", err)
			continue
		}

		for _, rj := range rawJobs {
			if rj.Title == "" || rj.URL == "" {
				continue
			}

			country := inferCountryNaukri(rj.Location)
			desc := rj.Description
			if desc == "" {
				desc = rj.Title + " at " + rj.Company
			}
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}

			postedAt := parseNaukriDate(rj.DateText)

			job := &models.Job{
				Title:       rj.Title,
				Company:     rj.Company,
				SourceURL:   rj.URL,
				SourceBoard: "naukri",
				Description: desc,
				Location:    rj.Location,
				Country:     country,
				JobType:     "Full Time",
				PostedAt:    postedAt,
			}

			NormalizeJob(job)
			if !seen[job.JobHash] {
				seen[job.JobHash] = true
				allJobs = append(allJobs, *job)
			}
		}
	}

	log.Printf("[Scraper] Naukri: scraped %d unique jobs", len(allJobs))
	return allJobs, nil
}

func inferCountryNaukri(location string) string {
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

	return "India" // Naukri is India-centric by default
}

func parseNaukriDate(dateText string) *time.Time {
	dateText = strings.ToLower(dateText)
	now := time.Now()

	if strings.Contains(dateText, "today") || strings.Contains(dateText, "just now") || strings.Contains(dateText, "hour") {
		return &now
	}

	if strings.Contains(dateText, "yesterday") || strings.Contains(dateText, "1 day ago") {
		t := now.AddDate(0, 0, -1)
		return &t
	}

	// Example: "3 days ago", "1 week ago", "3+ weeks ago"
	var days int
	if strings.Contains(dateText, "week") {
		days = 7
		if strings.Contains(dateText, "2") {
			days = 14
		} else if strings.Contains(dateText, "3") {
			days = 21
		}
	} else {
		// Try to parse number of days
		for d := 2; d <= 30; d++ {
			if strings.Contains(dateText, fmt.Sprintf("%d day", d)) {
				days = d
				break
			}
		}
	}

	if days > 0 {
		t := now.AddDate(0, 0, -days)
		return &t
	}

	return nil
}
