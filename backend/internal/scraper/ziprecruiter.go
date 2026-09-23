package scraper

import (
	"encoding/json"
	"log"
	"remotehunter/internal/models"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type ZipRecruiterScraper struct{}

func NewZipRecruiterScraper() *ZipRecruiterScraper {
	return &ZipRecruiterScraper{}
}

func (s *ZipRecruiterScraper) Name() string { return "ziprecruiter" }

func (s *ZipRecruiterScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.ziprecruiter.com/Jobs/Golang?location=Remote"
	}

	ctx, cancel := newHeadlessContext(120 * time.Second)
	defer cancel()

	urls := strings.Split(targetURL, "|")
	var allJobs []models.Job
	var countries []string
	seen := make(map[string]bool)

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		log.Printf("[Scraper] ZipRecruiter: Navigating to %s", u)
		var jobsJSON string
		err := chromedp.Run(ctx,
			chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
			chromedp.Navigate(u),
			chromedp.Sleep(8*time.Second),
			chromedp.Evaluate(`
(function() {
  const cards = document.querySelectorAll('article');
  const results = [];
  cards.forEach(card => {
    const titleEl = card.querySelector('h2');
    if (!titleEl) return;
    const title = titleEl.textContent.trim();

    const linkEl = card.querySelector('a[href*="/Job/"]');
    if (!linkEl) return;
    const href = linkEl.getAttribute('href');
    const jobURL = href ? new URL(href, window.location.href).href : '';

    const companyEl = card.querySelector('a[href*="/co/"]') || card.querySelector('a[href*="/c/"]');
    const company = companyEl ? companyEl.textContent.trim() : 'N/A';

    const locEl = card.querySelector('a[href*="location="]');
    const location = locEl ? locEl.textContent.trim() : 'Remote';

    const descEl = card.querySelector('p[class*="text-secondary"]');
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
			log.Printf("[Scraper] ZipRecruiter: chromedp scrape error for %s: %v", u, err)
			continue
		}

		if jobsJSON == "" || jobsJSON == "[]" {
			var pageTitle, bodySnippet string
			_ = chromedp.Run(ctx,
				chromedp.Evaluate(`document.title`, &pageTitle),
				chromedp.Evaluate(`document.body ? document.body.innerText.substring(0, 500) : "No body"`, &bodySnippet),
			)
			log.Printf("[Scraper] ZipRecruiter DEBUG: No jobs found. Page Title: '%s', Content: '%s'", pageTitle, bodySnippet)
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
			log.Printf("[Scraper] ZipRecruiter: failed to parse JSON: %v", err)
			continue
		}

		for _, rj := range rawJobs {
			if rj.Title == "" || rj.URL == "" {
				continue
			}

			country := inferCountryZipRecruiter(rj.Location)
			countries = append(countries, rj.Location+" => "+country)
			desc := rj.Description
			if desc == "Estimated pay" || desc == "" {
				desc = rj.Title + " at " + rj.Company
			}
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}

			job := &models.Job{
				Title:       rj.Title,
				Company:     rj.Company,
				SourceURL:   rj.URL,
				SourceBoard: "ziprecruiter",
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

	log.Printf("[Scraper] ZipRecruiter: scraped %d unique jobs (%s)", len(allJobs), strings.Join(countries, ", "))
	return allJobs, nil
}

func inferCountryZipRecruiter(location string) string {
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

	if strings.Contains(loc, "canada") || hasTrailingRegionCode(loc, canadaProvinceCodes) {
		return "Canada"
	}

	if strings.Contains(loc, "usa") || strings.Contains(loc, "united states") || strings.Contains(loc, "us only") ||
		hasTrailingRegionCode(loc, usStateCodes) {
		return "US"
	}

	// ponytail: an unrecognised location still falls through to "US" because
	// ZipRecruiter is US-centric and the scheduler drops US rows. Teach this
	// function the country before pointing the board at a non-US search.
	return "US"
}

// Two-letter region codes. Matching a bare "ca" substring instead (as this
// function used to) labels Chicago, Cary and "Fremont, CA" all the same way.
const (
	usStateCodes        = "al ak az ar ca co ct de dc fl ga hi id il in ia ks ky la me md ma mi mn ms mo mt ne nv nh nj nm ny nc nd oh ok or pa ri sc sd tn tx ut vt va wa wv wi wy"
	canadaProvinceCodes = "on bc ab qc ns mb sk nb nl pe yt nt nu"
)

// hasTrailingRegionCode reports whether loc ends in ", xx" or " xx", where xx is
// one of the space-separated codes.
func hasTrailingRegionCode(loc, codes string) bool {
	if len(loc) < 4 {
		return false
	}
	if sep := loc[len(loc)-3]; sep != ',' && sep != ' ' {
		return false
	}
	return strings.Contains(" "+codes+" ", " "+loc[len(loc)-2:]+" ")
}
