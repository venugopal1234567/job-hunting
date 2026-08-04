package scraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"remotehunter/internal/models"
	"strings"
	"time"
)

// LinkedInScraper scrapes jobs from LinkedIn guest API search results
type LinkedInScraper struct {
	client *http.Client
}

func NewLinkedInScraper() *LinkedInScraper {
	return &LinkedInScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *LinkedInScraper) Name() string { return "linkedin" }

func (s *LinkedInScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=golang&location=India&f_WT=2"
	}

	// Split by | to support multiple search targets if configured
	urls := strings.Split(targetURL, "|")
	var allJobs []models.Job
	seen := make(map[string]bool)

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		// We will pull the first 2 pages (50 jobs total per query target)
		for page := 0; page < 2; page++ {
			start := page * 25
			pageURL, err := appendStartParam(u, start)
			if err != nil {
				log.Printf("[Scraper] LinkedIn: failed to parse url %s: %v", u, err)
				continue
			}

			jobs, err := s.scrapePage(pageURL)
			if err != nil {
				log.Printf("[Scraper] LinkedIn: error scraping page %s: %v", pageURL, err)
				break // stop paginating this target if we hit an error (e.g. rate limit)
			}

			if len(jobs) == 0 {
				break // no more jobs found
			}

			for _, job := range jobs {
				if !seen[job.JobHash] {
					seen[job.JobHash] = true
					allJobs = append(allJobs, job)
				}
			}

			// Polite delay between requests to avoid 429 rate limit
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("[Scraper] LinkedIn: scraped %d unique jobs", len(allJobs))
	return allJobs, nil
}

func (s *LinkedInScraper) scrapePage(pageURL string) ([]models.Job, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("HTTP 429: Rate limited by LinkedIn")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlContent := string(body)
	return parseLinkedInHTML(htmlContent)
}

func appendStartParam(rawURL string, start int) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := parsed.Query()
	q.Set("start", fmt.Sprintf("%d", start))
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func parseLinkedInHTML(htmlContent string) ([]models.Job, error) {
	// Job cards are typically list items <li> or divs starting with base-search-card
	// We will split the HTML by "<li" to isolate each job card element.
	parts := strings.Split(htmlContent, "<li")
	var jobs []models.Job

	reCardLink := regexp.MustCompile(`(?s)class="[^"]*base-card__full-link[^"]*"[^>]*href="([^"]+)"`)
	reTitle := regexp.MustCompile(`(?s)<span class="sr-only">\s*([\s\S]*?)\s*</span>`)
	
	// Company matches either inside nested anchor or directly as subtitle text
	reCompanyLink := regexp.MustCompile(`(?s)class="base-search-card__subtitle"[^>]*>[\s\S]*?<a[^>]*>([\s\S]*?)</a>`)
	reCompanyText := regexp.MustCompile(`(?s)class="base-search-card__subtitle"[^>]*>\s*([\s\S]*?)\s*</h4>`)
	
	reLocation := regexp.MustCompile(`(?s)class="job-search-card__location"[^>]*>\s*([\s\S]*?)\s*</span>`)
	reDate := regexp.MustCompile(`(?s)<time[^>]*datetime="([^"]+)"`)
	reDateText := regexp.MustCompile(`(?s)<time[^>]*>\s*([\s\S]*?)\s*</time>`)

	for _, part := range parts {
		if !strings.Contains(part, "base-search-card") {
			continue
		}

		// URL
		linkMatch := reCardLink.FindStringSubmatch(part)
		if len(linkMatch) < 2 {
			continue
		}
		jobURL := strings.Split(linkMatch[1], "?")[0]

		// Title
		titleMatch := reTitle.FindStringSubmatch(part)
		if len(titleMatch) < 2 {
			continue
		}
		title := strings.TrimSpace(titleMatch[1])

		// Company
		company := "LinkedIn Guest"
		if compLinkMatch := reCompanyLink.FindStringSubmatch(part); len(compLinkMatch) >= 2 {
			company = strings.TrimSpace(compLinkMatch[1])
		} else if compTextMatch := reCompanyText.FindStringSubmatch(part); len(compTextMatch) >= 2 {
			company = strings.TrimSpace(compTextMatch[1])
		}

		// Location
		location := "Remote"
		if locMatch := reLocation.FindStringSubmatch(part); len(locMatch) >= 2 {
			location = strings.TrimSpace(locMatch[1])
		}

		// Date posted
		var postedAt *time.Time
		if dateMatch := reDate.FindStringSubmatch(part); len(dateMatch) >= 2 {
			if t, err := time.Parse("2006-01-02", strings.TrimSpace(dateMatch[1])); err == nil {
				postedAt = &t
			}
		}
		if postedAt == nil {
			if dateTextMatch := reDateText.FindStringSubmatch(part); len(dateTextMatch) >= 2 {
				postedAt = parseLinkedInRelativeDate(strings.TrimSpace(dateTextMatch[1]))
			}
		}

		country := inferCountryLinkedIn(location)
		desc := fmt.Sprintf("%s position at %s (%s). Remote Golang Engineer role.", title, company, location)

		job := &models.Job{
			Title:       title,
			Company:     company,
			SourceURL:   jobURL,
			SourceBoard: "linkedin",
			Description: desc,
			Location:    location,
			Country:     country,
			JobType:     "Full Time",
			PostedAt:    postedAt,
		}

		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func inferCountryLinkedIn(location string) string {
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

	cities := []string{"bangalore", "bengaluru", "karnataka", "pune", "hyderabad", "chennai", "mumbai", "delhi", "noida", "gurgaon", "kolkata", "calcutta"}
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

func parseLinkedInRelativeDate(text string) *time.Time {
	text = strings.ToLower(text)
	now := time.Now()

	// Handle standard indicators
	if strings.Contains(text, "today") || strings.Contains(text, "just now") || strings.Contains(text, "hour") || strings.Contains(text, "minute") {
		return &now
	}
	if strings.Contains(text, "yesterday") {
		t := now.AddDate(0, 0, -1)
		return &t
	}

	// Match "X days ago", "X weeks ago", "X months ago"
	reNum := regexp.MustCompile(`(\d+)`)
	match := reNum.FindString(text)
	if match == "" {
		return nil
	}
	
	var val int
	fmt.Sscanf(match, "%d", &val)
	if val <= 0 {
		return nil
	}

	var t time.Time
	if strings.Contains(text, "day") {
		t = now.AddDate(0, 0, -val)
	} else if strings.Contains(text, "week") {
		t = now.AddDate(0, 0, -val*7)
	} else if strings.Contains(text, "month") {
		t = now.AddDate(0, -val, 0)
	} else {
		return nil
	}

	return &t
}
