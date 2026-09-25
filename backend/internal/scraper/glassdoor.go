package scraper

import (
	"encoding/json"
	"log"
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

	ctx, cancel := newHeadlessContext(120 * time.Second)
	defer cancel()

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

    // Glassdoor lists "Listed X days ago" in a time/span element
    const dateEl = card.querySelector('time') ||
                   card.querySelector('[data-test="job-age"]') ||
                   card.querySelector('.listed') ||
                   Array.from(card.querySelectorAll('span, time, div')).find(n => /listed|posted|ago|today|yesterday/i.test(n.textContent));
    const dateStr = dateEl ? (dateEl.dateTime || dateEl.getAttribute('datetime') || dateEl.textContent.trim()) : '';

    results.push({
      title: title,
      company: company,
      url: jobURL,
      location: location,
      description: description,
      posted_at: dateStr
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
			PostedAt    string `json:"posted_at"`
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

			if postedAt := parseRelativeDate(rj.PostedAt); postedAt != nil {
				job.PostedAt = postedAt
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
