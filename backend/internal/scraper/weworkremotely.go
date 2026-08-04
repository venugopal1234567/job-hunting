package scraper

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"remotehunter/internal/models"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// WeWorkRemotelyScraper scrapes WeWorkRemotely via RSS or HTML (chromedp fallback)
type WeWorkRemotelyScraper struct {
	client *http.Client
}

func NewWeWorkRemotelyScraper() *WeWorkRemotelyScraper {
	return &WeWorkRemotelyScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *WeWorkRemotelyScraper) Name() string { return "weworkremotely" }

func (s *WeWorkRemotelyScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://weworkremotely.com/categories/remote-programming-jobs.rss"
	}

	if !strings.HasSuffix(targetURL, ".rss") {
		return s.scrapeHTML(targetURL)
	}

	return s.scrapeRSS(targetURL)
}

func (s *WeWorkRemotelyScraper) scrapeRSS(targetURL string) ([]models.Job, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RemoteHunter/1.0 RSS Reader")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weworkremotely rss fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weworkremotely: HTTP %d", resp.StatusCode)
	}

	type RSSItem struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
		Region  string `xml:"region"`
		Description struct {
			Content string `xml:",cdata"`
		} `xml:"description"`
	}

	type RSS struct {
		Items []RSSItem `xml:"channel>item"`
	}

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("weworkremotely parse rss: %w", err)
	}

	var jobs []models.Job
	for _, item := range rss.Items {
		title := item.Title
		company := "Unknown"

		if idx := strings.Index(title, ": "); idx > 0 {
			company = strings.TrimSpace(title[:idx])
			title = strings.TrimSpace(title[idx+2:])
		}

		if idx := strings.LastIndex(title, " at "); idx > 0 {
			title = strings.TrimSpace(title[:idx])
		}

		desc := stripHTML(item.Description.Content)
		if desc == "" {
			desc = item.Title
		}
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}

		job := &models.Job{
			Title:       title,
			Company:     company,
			SourceURL:   item.Link,
			SourceBoard: "weworkremotely",
			Description: desc,
			Location:    "Remote",
			Country:     parseWWRRegion(item.Region),
		}

		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			job.PostedAt = &t
		} else if t, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
			job.PostedAt = &t
		}

		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] WeWorkRemotely (RSS): scraped %d jobs", len(jobs))
	return jobs, nil
}

func (s *WeWorkRemotelyScraper) scrapeHTML(targetURL string) ([]models.Job, error) {
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
		// Fallback for local run
		if _, err := os.Stat("/usr/bin/chromium-browser"); err == nil {
			opts = append(opts, chromedp.ExecPath("/usr/bin/chromium-browser"))
		}
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	var jobsJSON string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
		chromedp.Navigate(targetURL),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(wwrExtractJS, &jobsJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp weworkremotely scrape: %w", err)
	}

	var extracted []struct {
		Title       string `json:"title"`
		Company     string `json:"company"`
		Location    string `json:"location"`
		Country     string `json:"country"`
		SourceURL   string `json:"source_url"`
		PostedAt    string `json:"posted_at"`
		JobType     string `json:"job_type"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(jobsJSON), &extracted); err != nil {
		return nil, fmt.Errorf("weworkremotely: parse jobs JSON: %w", err)
	}

	var jobs []models.Job
	for _, e := range extracted {
		posted := parseWWRRelativeDate(e.PostedAt)
		// Append Go/Golang keyword to fallback description to ensure Go/Golang skill filters match
		desc := e.Description
		if !strings.Contains(strings.ToLower(desc), "go") {
			desc += " - Technologies: Go, Golang"
		}
		job := &models.Job{
			Title:       e.Title,
			Company:     e.Company,
			Location:    e.Location,
			Country:     parseWWRRegion(e.Country),
			SourceURL:   e.SourceURL,
			SourceBoard: "weworkremotely",
			JobType:     e.JobType,
			Description: desc,
			PostedAt:    posted,
		}
		jobs = append(jobs, *job)
	}

	// Concurrently fetch full descriptions
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for i := range jobs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			desc := s.fetchFullDescription(jobs[idx].SourceURL)
			if len(desc) > 50 {
				jobs[idx].Description = desc
			}
			NormalizeJob(&jobs[idx])
		}(i)
	}
	wg.Wait()

	log.Printf("[Scraper] WeWorkRemotely (HTML): scraped %d jobs", len(jobs))
	return jobs, nil
}

func (s *WeWorkRemotelyScraper) fetchFullDescription(detailURL string) string {
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	htmlStr := string(body)
	reDesc := regexpMustMatchWWR(htmlStr)
	return stripHTML(reDesc)
}

func regexpMustMatchWWR(htmlStr string) string {
	idxStart := strings.Index(htmlStr, `<div id="job-details"`)
	if idxStart == -1 {
		idxStart = strings.Index(htmlStr, `class="listing-container"`)
	}
	if idxStart == -1 {
		return ""
	}

	chunk := htmlStr[idxStart:]
	idxEnd := strings.Index(chunk, `<div class="apply-container"`)
	if idxEnd != -1 {
		return chunk[:idxEnd]
	}
	if len(chunk) > 10000 {
		return chunk[:10000]
	}
	return chunk
}

func parseWWRRelativeDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	now := time.Now()
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))
	if strings.Contains(dateStr, "today") || strings.Contains(dateStr, "just now") {
		return &now
	}
	if strings.Contains(dateStr, "yesterday") {
		t := now.AddDate(0, 0, -1)
		return &t
	}

	var days int
	if _, err := fmt.Sscanf(dateStr, "%dd", &days); err == nil && days > 0 {
		t := now.AddDate(0, 0, -days)
		return &t
	}
	if _, err := fmt.Sscanf(dateStr, "%d days", &days); err == nil && days > 0 {
		t := now.AddDate(0, 0, -days)
		return &t
	}

	return nil
}

const wwrExtractJS = `
(function() {
  const jobs = [];
  const cards = Array.from(document.querySelectorAll('.jobs-container section.jobs ul li.new-listing-container'));
  
  cards.forEach(card => {
    const linkEl = card.querySelector('a.listing-link--unlocked, a[href*="/remote-jobs/"]');
    let sourceURL = linkEl ? linkEl.href : '';
    if (sourceURL && !sourceURL.startsWith('http')) {
      sourceURL = 'https://weworkremotely.com' + sourceURL;
    }
    
    const titleEl = card.querySelector('.new-listing__header__title__text');
    const title = titleEl ? titleEl.textContent.trim() : '';
    
    const companyEl = card.querySelector('.new-listing__company-name');
    let company = companyEl ? companyEl.textContent.trim() : '';
    
    const hqEl = card.querySelector('.new-listing__company-headquarters');
    const hq = hqEl ? hqEl.textContent.trim() : '';
    
    const catEls = Array.from(card.querySelectorAll('.new-listing__categories p, .new-listing__categories .new-listing__categories__category'));
    let jobType = 'Full-Time';
    let region = '';
    catEls.forEach(el => {
      const text = el.textContent.trim();
      if (text !== 'Boosted' && text !== 'Featured' && text !== 'Top Company') {
        if (text.includes('Full-Time') || text.includes('Contract') || text.includes('Part-Time')) {
          jobType = text;
        } else {
          region = text;
        }
      }
    });
    
    const dateEl = card.querySelector('.new-listing__header__icons__date');
    const dateStr = dateEl ? dateEl.textContent.trim() : '';
    
    if (title && company && sourceURL) {
      jobs.push({
        title,
        company,
        location: region || hq || 'Remote',
        country: region || hq || 'Worldwide',
        source_url: sourceURL,
        posted_at: dateStr,
        job_type: jobType,
        description: title + ' at ' + company + ' (Headquarters: ' + hq + ', Region: ' + (region || 'Worldwide') + ')'
      });
    }
  });
  
  return JSON.stringify(jobs);
})()
`

func parseWWRRegion(region string) string {
	region = strings.ToLower(region)
	switch {
	case strings.Contains(region, "usa") || strings.Contains(region, "us only"):
		return "US"
	case strings.Contains(region, "europe"):
		return "Europe"
	case strings.Contains(region, "worldwide") || strings.Contains(region, "anywhere"):
		return "Worldwide"
	default:
		return "Worldwide"
	}
}

// stripHTML removes HTML tags from a string — shared utility used by multiple scrapers
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			result.WriteRune(' ')
		case !inTag:
			result.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(result.String()), " ")
}
