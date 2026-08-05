package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"remotehunter/internal/models"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"net/url"
)

// GoogleJobsScraper scrapes job listings from Google Jobs search results (udm=8).
// It uses chromedp headless browser because Google renders job cards with JavaScript.
type GoogleJobsScraper struct{}

func NewGoogleJobsScraper() *GoogleJobsScraper {
	return &GoogleJobsScraper{}
}

func (s *GoogleJobsScraper) Name() string { return "googlejobs" }

// Scrape navigates to Google Jobs search results and extracts job listings.
// The targetURL can contain multiple URLs separated by "|" (pipe) to scrape
// multiple searches in a single pass.
func (s *GoogleJobsScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("googlejobs: no target URL provided")
	}

	// 1. Check if SerpAPI key is available in the environment
	apiKey := os.Getenv("SERPAPI_API_KEY")
	if apiKey != "" {
		log.Println("[Scraper] GoogleJobs: SerpAPI key found. Using SerpAPI for reliable search results.")
		return s.scrapeWithSerpAPI(targetURL, apiKey)
	}

	log.Println("[Scraper] GoogleJobs: No SERPAPI_API_KEY found. Falling back to chromedp (susceptible to CAPTCHAs).")

	// Support multiple URLs separated by pipe
	urls := strings.Split(targetURL, "|")

	var allJobs []models.Job
	seen := make(map[string]bool)

	for _, rawURL := range urls {
		u := strings.TrimSpace(rawURL)
		if u == "" {
			continue
		}

		jobs, err := s.scrapeOneURL(u)
		if err != nil {
			log.Printf("[Scraper] GoogleJobs: error scraping %s: %v", u, err)
			continue
		}

		for _, job := range jobs {
			if !seen[job.JobHash] {
				seen[job.JobHash] = true
				allJobs = append(allJobs, job)
			}
		}
	}

	log.Printf("[Scraper] GoogleJobs: scraped %d unique jobs from %d URL(s)", len(allJobs), len(urls))
	return allJobs, nil
}

func (s *GoogleJobsScraper) scrapeWithSerpAPI(targetURL string, apiKey string) ([]models.Job, error) {
	urls := strings.Split(targetURL, "|")
	var allJobs []models.Job
	seen := make(map[string]bool)

	for _, rawURL := range urls {
		u := strings.TrimSpace(rawURL)
		if u == "" {
			continue
		}

		// Extract the query parameter 'q' from the Google Search URL
		qVal := "Senior Golang remote jobs"
		if strings.Contains(u, "q=") {
			parts := strings.Split(u, "q=")
			if len(parts) > 1 {
				qPart := strings.Split(parts[1], "&")[0]
				if decoded, err := url.QueryUnescape(qPart); err == nil {
					qVal = decoded
				} else {
					qVal = strings.ReplaceAll(qPart, "+", " ")
				}
			}
		}

		// We call the SerpAPI endpoint directly via HTTP client
		apiURL := fmt.Sprintf("https://serpapi.com/search.json?engine=google_jobs&q=%s&api_key=%s", url.QueryEscape(qVal), apiKey)
		log.Printf("[Scraper] GoogleJobs DEBUG: SerpAPI Query: '%s'", qVal)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			log.Printf("[Scraper] GoogleJobs: failed to create SerpAPI request: %v", err)
			continue
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Scraper] GoogleJobs: SerpAPI request failed: %v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[Scraper] GoogleJobs: SerpAPI returned status %d", resp.StatusCode)
			continue
		}

		var results struct {
			JobsResults []struct {
				Title              string `json:"title"`
				CompanyName        string `json:"company_name"`
				Location           string `json:"location"`
				Description        string `json:"description"`
				ShareLink          string `json:"share_link"`
				DetectedExtensions struct {
					PostedAt string `json:"posted_at"`
				} `json:"detected_extensions"`
			} `json:"jobs_results"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			log.Printf("[Scraper] GoogleJobs: failed to decode SerpAPI JSON: %v", err)
			continue
		}
		log.Printf("[Scraper] GoogleJobs DEBUG: SerpAPI returned %d jobs", len(results.JobsResults))

		for _, item := range results.JobsResults {
			title := item.Title
			company := item.CompanyName
			location := item.Location
			if location == "" {
				location = "Remote"
			}
			desc := item.Description
			if desc == "" {
				desc = fmt.Sprintf("%s at %s (%s)", title, company, location)
			}
			link := item.ShareLink
			if link == "" {
				link = u
			}

			board := "googlejobs"
			if strings.Contains(u, "board=googlejobscompanylist") {
				board = "googlejobscompanylist"
			}

			job := models.Job{
				Title:       title,
				Company:     company,
				Location:    location,
				Country:     inferCountryGoogle(location, u),
				SourceURL:   link,
				SourceBoard: board,
				Description: desc,
				PostedAt:    parseRelativeDate(item.DetectedExtensions.PostedAt),
			}
			NormalizeJob(&job)

			if !seen[job.JobHash] {
				seen[job.JobHash] = true
				allJobs = append(allJobs, job)
			}
		}
	}

	return allJobs, nil
}

func (s *GoogleJobsScraper) scrapeOneURL(targetURL string) ([]models.Job, error) {
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
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	// Inject script to delete navigator.webdriver to bypass detection
	var jobsJSON string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
		chromedp.Navigate(targetURL),
		chromedp.Sleep(6*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 3)`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 2)`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(googleJobsExtractJS, &jobsJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("googlejobs chromedp: %w", err)
	}

	return parseGoogleJobs(jobsJSON, targetURL)
}

// googleJobsExtractJS is the JavaScript that runs in the browser context to
// extract job card data from Google Jobs search results.
const googleJobsExtractJS = `
(function() {
  const jobs = [];
  const seen = new Set();

  // Google Jobs renders results in li elements within the job listing container.
  // The exact selectors may change, so we try multiple strategies.
  
  const jobCards = document.querySelectorAll('li.iFjolb, div.PwjeAc, li[data-entityid], div[jscontroller] li[class]');

  jobCards.forEach(card => {
    // Try to extract the title
    const titleEl = card.querySelector('div.BjJfJf, span.sH3zle, h3, div[role="heading"], .tNxQIb');
    let title = (titleEl ? titleEl.textContent : '').trim();
    // Extract company name
    const companyEl = card.querySelector('div.vNEEBe, span.nJlDiv, .company, div[class*="company"]');
    let company = (companyEl ? companyEl.textContent : '').trim();
    if (!company) {
      // Try to get from second line of text
      const lines = card.querySelectorAll('div[class]');
      for (const line of lines) {
        const text = line.textContent.trim();
        if (text && text !== title && text.length > 1 && text.length < 100 && !text.includes('day') && !text.includes('ago')) {
          company = text;
          break;
        }
      }
    }
    if (!company) company = 'Unknown';

    // Extract location
    const locEl = card.querySelector('div.Qk80Jf, span[class*="loc"], .location');
    let location = (locEl ? locEl.textContent : '').trim();
    if (!location) location = 'Remote';

    // Extract source URL (the link to the job detail)
    let sourceURL = '';
    const linkEl = card.querySelector('a[href]');
    if (linkEl) {
      sourceURL = linkEl.href || '';
    }

    // Extract any date info
    const dateEl = card.querySelector('span[class*="LL4CDc"], span.SuWscb, .date');
    const dateStr = (dateEl ? dateEl.textContent : '').trim();

    const key = title + '|' + company;
    if (!seen.has(key)) {
      seen.add(key);
      jobs.push({
        title: title,
        company: company,
        location: location,
        source_url: sourceURL,
        posted_at: dateStr,
        description: title + ' at ' + company + (location ? ' (' + location + ')' : '')
      });
    }
  });

  // Strategy 2: If we didn't find jobs with strategy 1, try broader selectors
  if (jobs.length === 0) {
    // Look for the Google Jobs widget results
    const allLinks = document.querySelectorAll('a[jsname], div[jsaction] a');
    
    allLinks.forEach(link => {
      const container = link.closest('li') || link.closest('div[jscontroller]') || link.parentElement;
      if (!container) return;

      const texts = [];
      container.querySelectorAll('div, span').forEach(el => {
        const t = el.textContent.trim();
        if (t && t.length > 2 && t.length < 200 && !texts.includes(t)) {
          texts.push(t);
        }
      });

      if (texts.length < 2) return;

      const title = texts[0];
      const company = texts[1] || 'Unknown';
      const location = texts.length > 2 ? texts[2] : 'Remote';
      const dateStr = texts.length > 3 ? texts[texts.length - 1] : '';

      const sourceURL = link.href || '';
      const key = title + '|' + company;
      
      if (title.length > 3 && !seen.has(key)) {
        seen.add(key);
        jobs.push({
          title: title,
          company: company,
          location: location,
          source_url: sourceURL,
          posted_at: dateStr,
          description: title + ' at ' + company + (location ? ' (' + location + ')' : '')
        });
      }
    });
  }

  // Strategy 3: Try clicking on individual job cards to get detail info
  // (This populates the detail panel on the right side)
  const detailPanel = document.querySelector('div#tl_ditsc, div.whazf, div[id*="job_details"]');
  if (detailPanel) {
    const descText = detailPanel.textContent || '';
    if (descText.length > 50 && jobs.length > 0) {
      // Attach the visible detail to the first job
      jobs[0].description = descText.substring(0, 3000);
    }
  }

  return JSON.stringify(jobs);
})()
`

func parseGoogleJobs(jobsJSON, fallbackURL string) ([]models.Job, error) {
	var extracted []struct {
		Title       string `json:"title"`
		Company     string `json:"company"`
		Location    string `json:"location"`
		SourceURL   string `json:"source_url"`
		PostedAt    string `json:"posted_at"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(jobsJSON), &extracted); err != nil {
		return nil, fmt.Errorf("googlejobs: parse JSON: %w", err)
	}

	var jobs []models.Job
	for _, e := range extracted {
		title := strings.TrimSpace(e.Title)
		if title == "" || len(title) < 3 {
			continue
		}

		company := strings.TrimSpace(e.Company)
		if company == "" {
			company = "Unknown"
		}

		sourceURL := e.SourceURL
		if sourceURL == "" {
			sourceURL = fallbackURL
		}

		location := strings.TrimSpace(e.Location)
		if location == "" {
			location = "Remote"
		}

		country := inferCountryGoogle(location, fallbackURL)

		desc := strings.TrimSpace(e.Description)
		if desc == "" {
			desc = fmt.Sprintf("%s at %s (%s)", title, company, location)
		}

		board := "googlejobs"
		if strings.Contains(fallbackURL, "board=googlejobscompanylist") {
			board = "googlejobscompanylist"
		}

		job := &models.Job{
			Title:       title,
			Company:     company,
			Location:    location,
			Country:     country,
			SourceURL:   sourceURL,
			SourceBoard: board,
			Description: desc,
			PostedAt:    parseRelativeDate(e.PostedAt),
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] GoogleJobs: parsed %d jobs", len(jobs))
	return jobs, nil
}

// inferCountryGoogle determines country from location text and the search query context.
func inferCountryGoogle(location, searchURL string) string {
	loc := strings.ToLower(location)

	// Check location string first
	switch {
	case strings.Contains(loc, "india") || strings.Contains(loc, "bangalore") ||
		strings.Contains(loc, "bengaluru") || strings.Contains(loc, "hyderabad") ||
		strings.Contains(loc, "chennai") || strings.Contains(loc, "pune") ||
		strings.Contains(loc, "mumbai") || strings.Contains(loc, "delhi") ||
		strings.Contains(loc, "noida") || strings.Contains(loc, "gurgaon") ||
		strings.Contains(loc, "karnataka"):
		return "India"
	case strings.Contains(loc, "worldwide") || strings.Contains(loc, "anywhere") ||
		strings.Contains(loc, "global") || loc == "remote":
		return "Worldwide"
	case strings.Contains(loc, "usa") || strings.Contains(loc, "united states") ||
		strings.Contains(loc, "us only"):
		return "US"
	case strings.Contains(loc, "europe") || strings.Contains(loc, "eu only"):
		return "Europe"
	case strings.Contains(loc, "uk") || strings.Contains(loc, "united kingdom"):
		return "UK"
	}

	// Infer from search query context
	searchLower := strings.ToLower(searchURL)
	if strings.Contains(searchLower, "india") || strings.Contains(searchLower, "worldwide") ||
		strings.Contains(searchLower, "anywhere") {
		return "Worldwide"
	}

	if strings.Contains(loc, "remote") {
		return "Worldwide"
	}

	return location
}
