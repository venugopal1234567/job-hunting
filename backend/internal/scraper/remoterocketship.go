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

// RemoteRocketshipScraper scrapes jobs from Remote Rocketship using chromedp.
type RemoteRocketshipScraper struct{}

func NewRemoteRocketshipScraper() *RemoteRocketshipScraper {
	return &RemoteRocketshipScraper{}
}

func (s *RemoteRocketshipScraper) Name() string { return "remoterocketship" }

func (s *RemoteRocketshipScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.remoterocketship.com/?ref=yanirs-established-remote&page=1&sort=DateAdded&jobTitle=Golang&locations=Worldwide%2CIndia"
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

	// Respect CHROME_PATH if set, otherwise default to typical /usr/bin/chromium-browser or /usr/bin/google-chrome
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

	var jobsJSON string
	log.Printf("[Scraper] RemoteRocketship: Navigating to %s", targetURL)

	err := chromedp.Run(ctx,
		chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
		chromedp.Navigate(targetURL),
		chromedp.Sleep(10*time.Second), // wait for CF managed challenge & JS execution
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 3)`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
(function() {
  const cards = document.querySelectorAll('div.bg-secondary');
  const results = [];
  cards.forEach(card => {
    const titleEl = card.querySelector('h3 a');
    if (!titleEl) return;
    const title = titleEl.textContent.trim();
    const relativeURL = titleEl.getAttribute('href');
    const jobURL = relativeURL ? new URL(relativeURL, window.location.href).href : '';

    const companyEl = card.querySelector('h4 a');
    const company = companyEl ? companyEl.textContent.trim() : '';

    const descEl = card.querySelector('p.text-secondary.mb-4.text-sm') || card.querySelector('p.text-sm');
    const description = descEl ? descEl.textContent.trim() : '';

    // Extract all badges/pills
    const pills = [];
    card.querySelectorAll('p').forEach(p => {
      const text = p.textContent.trim();
      if (text && !text.startsWith('🕒') && !text.includes('Loved by')) {
        pills.push(text);
      }
    });

    // Extract posted time
    let postedTime = '';
    card.querySelectorAll('p').forEach(p => {
      const text = p.textContent.trim();
      if (text.startsWith('🕒')) {
        postedTime = text.replace('🕒', '').trim();
      }
    });

    results.push({
      title: title,
      company: company,
      url: jobURL,
      description: description,
      pills: pills,
      postedTime: postedTime
    });
  });
  return JSON.stringify(results);
})()
		`, &jobsJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("remoterocketship chromedp: %w", err)
	}

	type rawJob struct {
		Title       string   `json:"title"`
		Company     string   `json:"company"`
		URL         string   `json:"url"`
		Description string   `json:"description"`
		Pills       []string `json:"pills"`
		PostedTime  string   `json:"postedTime"`
	}

	var rawJobs []rawJob
	if err := json.Unmarshal([]byte(jobsJSON), &rawJobs); err != nil {
		return nil, fmt.Errorf("remoterocketship unmarshal: %w", err)
	}

	var jobs []models.Job
	for _, rj := range rawJobs {
		if rj.Title == "" || rj.Company == "" || rj.URL == "" {
			continue
		}

		// Find location from pills
		location := "Remote"
		for _, pill := range rj.Pills {
			if strings.Contains(pill, "Remote") || strings.Contains(pill, "Anywhere") || strings.Contains(pill, "Worldwide") {
				location = pill
				break
			}
		}

		country := inferCountryRemoteRocketship(location)
		postedAt := parseRemoteRocketshipDate(rj.PostedTime)

		// Job type (default is Full-time, otherwise extract from pills if found)
		jobType := "Full Time"
		for _, pill := range rj.Pills {
			if strings.Contains(pill, "Full Time") || strings.Contains(pill, "Contract") || strings.Contains(pill, "Internship") || strings.Contains(pill, "Part Time") {
				jobType = strings.Replace(pill, "⏰", "", -1)
				jobType = strings.TrimSpace(jobType)
				break
			}
		}

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
			SourceBoard: "remoterocketship",
			Description: desc,
			Location:    location,
			Country:     country,
			JobType:     jobType,
			PostedAt:    postedAt,
		}

		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] RemoteRocketship: fetched %d jobs", len(jobs))
	return jobs, nil
}

func inferCountryRemoteRocketship(location string) string {
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

	cities := []string{"bangalore", "bengaluru", "karnataka", "pune", "hyderabad", "hyderābād", "chennai", "mumbai", "delhi", "noida", "gurgaon", "kolkata", "calcutta"}
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

func parseRemoteRocketshipDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	dateStr = strings.TrimSpace(dateStr)

	var parsed time.Time
	var err error

	// If it contains a year (four digits)
	hasYear := false
	for _, y := range []string{"2024", "2025", "2026", "2027"} {
		if strings.Contains(dateStr, y) {
			hasYear = true
			break
		}
	}

	if hasYear {
		// Try standard formats with year
		formats := []string{
			"January 2, 2006",
			"Jan 2, 2006",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err = time.Parse(f, dateStr); err == nil {
				return &parsed
			}
		}
	} else {
		// Missing year. Append current year.
		currentYear := time.Now().Year()
		dateWithYear := fmt.Sprintf("%s, %d", dateStr, currentYear)
		formats := []string{
			"January 2, 2006",
			"Jan 2, 2006",
		}
		for _, f := range formats {
			if parsed, err = time.Parse(f, dateWithYear); err == nil {
				// If parsed date is in the future compared to now (e.g., today is Jan and date says Dec), it might be last year
				if parsed.After(time.Now()) {
					parsed = parsed.AddDate(-1, 0, 0)
				}
				return &parsed
			}
		}
	}
	return nil
}
