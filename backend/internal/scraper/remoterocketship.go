package scraper

import (
	"encoding/json"
	"fmt"
	"log"
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

	ctx, cancel := newHeadlessContext(120 * time.Second)
	defer cancel()

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

	// Card labels come in three shapes: "September 15" (no year), "Sep 15, 2026"
	// (with year) and a relative "6 days ago". time.Parse leaves the year at 0
	// when the format omits it, so one format list covers both absolute shapes.
	formats := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"January 2",
		"Jan 2",
		"2006-01-02",
	}
	for _, f := range formats {
		parsed, err := time.Parse(f, dateStr)
		if err != nil {
			continue
		}
		if parsed.Year() == 0 {
			// Year absent from the label: assume the current year, and roll back
			// when that lands in the future (a Dec date read in Jan is last year).
			parsed = parsed.AddDate(time.Now().Year(), 0, 0)
			if parsed.After(time.Now()) {
				parsed = parsed.AddDate(-1, 0, 0)
			}
		}
		return &parsed
	}

	// Relative labels ("6 days ago") reuse the shared parser rather than
	// returning nil, which the jobs query would bucket as freshly scraped.
	if rel := parseRelativeDate(dateStr); rel != nil {
		return rel
	}

	return nil
}
