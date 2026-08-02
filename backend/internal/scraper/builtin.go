package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"remotehunter/internal/models"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// BuiltInScraper scrapes JS-rendered & server-rendered job listings from BuiltIn
type BuiltInScraper struct{}

func NewBuiltInScraper() *BuiltInScraper {
	return &BuiltInScraper{}
}

func (s *BuiltInScraper) Name() string { return "builtin" }

func (s *BuiltInScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://builtin.com/jobs/remote/senior?search=Go&daysSinceUpdated=3&skills=Go%2CPython%2CAWS%2CDocker%2CGCP%2CTypescript%2CAzure%2CCi%2FCd%2CPostgres%2CRust%2CSQL%2CNode.js&country=IND&allLocations=true"
	} else if !strings.Contains(targetURL, "daysSinceUpdated") {
		if strings.Contains(targetURL, "?") {
			targetURL += "&daysSinceUpdated=3"
		} else {
			targetURL += "?daysSinceUpdated=3"
		}
	}

	// 1. Try direct HTTP GET parsing first (fast & reliable)
	httpJobs, err := scrapeBuiltInHTTP(targetURL)
	if err == nil && len(httpJobs) > 0 {
		log.Printf("[Scraper] BuiltIn (HTTP): scraped %d jobs with full details and relative dates", len(httpJobs))
		return httpJobs, nil
	}

	// 2. Fallback to chromedp headless browser
	log.Printf("[Scraper] BuiltIn HTTP returned 0 jobs (err: %v); running chromedp fallback...", err)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("window-size", "1920,1080"),
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
	)

	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 90*time.Second)
	defer cancelTimeout()

	var jobsJSON string
	runErr := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.Sleep(6*time.Second),
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight / 2)`, nil),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(builtInExtractJS, &jobsJSON),
	)

	if runErr != nil {
		return nil, fmt.Errorf("chromedp builtin scrape: %w", runErr)
	}

	return parseBuiltInJobs(jobsJSON, targetURL)
}

func scrapeBuiltInHTTP(targetURL string) ([]models.Job, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlStr := string(body)
	parts := strings.Split(htmlStr, `data-id="job-card"`)

	reTitle := regexp.MustCompile(`data-id="job-card-title"[^>]*>([^<]+)</a>`)
	reComp := regexp.MustCompile(`data-id="company-title"[^>]*>(?:<span[^>]*>)?([^<]+)|alt="([^"]+) Logo"`)
	reLink := regexp.MustCompile(`href="(/job/[^"]+)"`)
	reLoc := regexp.MustCompile(`<span[^>]*class="font-barlow[^"]*"[^>]*>([^<]+)</span>`)
	reDesc := regexp.MustCompile(`(?s)<div[^>]*class="fs-sm[^"]*"[^>]*>([\s\S]*?)</div>`)
	reDate := regexp.MustCompile(`(?i)(?:Job\s+Posted\s+|Reposted\s+)?(\d+\+?\s*(?:minute|hour|day|week|month|year)s?\s*ago|today|yesterday|just now)`)

	var jobs []models.Job
	seen := make(map[string]bool)

	for _, card := range parts[1:] {
		titleMatch := reTitle.FindStringSubmatch(card)
		if len(titleMatch) < 2 {
			continue
		}
		title := strings.TrimSpace(titleMatch[1])

		company := "BuiltIn Company"
		compMatch := reComp.FindStringSubmatch(card)
		if len(compMatch) >= 2 && compMatch[1] != "" {
			company = strings.TrimSpace(compMatch[1])
		} else if len(compMatch) >= 3 && compMatch[2] != "" {
			company = strings.TrimSpace(compMatch[2])
		}

		link := targetURL
		linkMatch := reLink.FindStringSubmatch(card)
		if len(linkMatch) >= 2 {
			link = "https://www.builtin.com" + linkMatch[1]
		}

		location := "Remote"
		locMatch := reLoc.FindStringSubmatch(card)
		if len(locMatch) >= 2 {
			location = strings.TrimSpace(locMatch[1])
		}

		desc := fmt.Sprintf("%s at %s (%s)", title, company, location)
		descMatch := reDesc.FindStringSubmatch(card)
		if len(descMatch) >= 2 {
			d := stripHTML(descMatch[1])
			if len(d) > 15 {
				desc = d
			}
		}

		// Extract accurate posted date
		var postedAt *time.Time
		dateMatch := reDate.FindStringSubmatch(card)
		if len(dateMatch) >= 1 {
			postedAt = parseRelativeDate(dateMatch[0])
		}

		key := link
		if seen[key] {
			continue
		}
		seen[key] = true

		country := inferCountry(location)
		if (country == "" || country == "Worldwide") && (strings.Contains(targetURL, "country=IND") || strings.Contains(targetURL, "country=India")) {
			country = "India"
		}

		job := &models.Job{
			Title:       title,
			Company:     company,
			Location:    location,
			Country:     country,
			SourceURL:   link,
			SourceBoard: "builtin",
			Description: desc,
			PostedAt:    postedAt,
		}
		jobs = append(jobs, *job)
	}

	// Concurrently fetch full detail page descriptions, dates & direct application links
	detailClient := &http.Client{Timeout: 10 * time.Second}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i := range jobs {
		if jobs[i].SourceURL == "" || !strings.HasPrefix(jobs[i].SourceURL, "http") {
			NormalizeJob(&jobs[i])
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullDesc, applyURL, detailDate := fetchBuiltInFullDetail(detailClient, jobs[idx].SourceURL)
			if len(fullDesc) > 50 {
				jobs[idx].Description = fullDesc
			}
			if applyURL != "" {
				jobs[idx].SourceURL = applyURL
			}
			if jobs[idx].PostedAt == nil && detailDate != nil {
				jobs[idx].PostedAt = detailDate
			}
			NormalizeJob(&jobs[idx])
		}(i)
	}
	wg.Wait()

	return jobs, nil
}

func parseRelativeDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	now := time.Now()
	lower := strings.ToLower(strings.TrimSpace(dateStr))

	if strings.Contains(lower, "today") || strings.Contains(lower, "just now") {
		return &now
	}
	if strings.Contains(lower, "yesterday") {
		t := now.AddDate(0, 0, -1)
		return &t
	}

	re := regexp.MustCompile(`(\d+)\+?\s*(minute|hour|day|week|month|year)s?\s*ago`)
	m := re.FindStringSubmatch(lower)
	if len(m) >= 3 {
		num, err := strconv.Atoi(m[1])
		if err == nil && num > 0 {
			var t time.Time
			switch m[2] {
			case "minute":
				t = now.Add(-time.Duration(num) * time.Minute)
			case "hour":
				t = now.Add(-time.Duration(num) * time.Hour)
			case "day":
				t = now.AddDate(0, 0, -num)
			case "week":
				t = now.AddDate(0, 0, -num*7)
			case "month":
				t = now.AddDate(0, -num, 0)
			case "year":
				t = now.AddDate(-num, 0, 0)
			}
			return &t
		}
	}

	return nil
}

func fetchBuiltInFullDetail(client *http.Client, detailURL string) (string, string, *time.Time) {
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", "", nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", "", nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil
	}

	htmlStr := string(body)

	applyLink := ""
	reApply := regexp.MustCompile(`"howToApply":\s*"([^"]+)"`)
	if m := reApply.FindStringSubmatch(htmlStr); len(m) >= 2 {
		applyLink = strings.ReplaceAll(m[1], `\/`, `/`)
	}

	var detailDate *time.Time
	reJSONDate := regexp.MustCompile(`(?i)"datePosted"\s*:\s*"(\d{4}-\d{2}-\d{2})`)
	if m := reJSONDate.FindStringSubmatch(htmlStr); len(m) >= 2 {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			detailDate = &t
		}
	}
	if detailDate == nil {
		reDate := regexp.MustCompile(`(?i)(?:Job\s+Posted\s+|Reposted\s+)(\d+\+?\s*(?:minute|hour|day|week|month|year)s?\s*ago|today|yesterday|just now)`)
		if m := reDate.FindStringSubmatch(htmlStr); len(m) >= 1 {
			detailDate = parseRelativeDate(m[0])
		}
	}

	// Strip script and style tags
	reScript := regexp.MustCompile(`(?s)<script[\s\S]*?</script>|<style[\s\S]*?</style>`)
	cleanHTML := reScript.ReplaceAllString(htmlStr, "")

	// Replace structural tags with newlines
	reBreak := regexp.MustCompile(`(?i)<br\s*/?>|</p>|div>|</li>|</h[1-6]>`)
	cleanHTML = reBreak.ReplaceAllString(cleanHTML, "\n")

	// Strip remaining HTML tags
	reTag := regexp.MustCompile(`<[^>]+>`)
	text := reTag.ReplaceAllString(cleanHTML, " ")

	lines := strings.Split(text, "\n")
	var validLines []string
	seen := make(map[string]bool)

	for _, line := range lines {
		l := html.UnescapeString(strings.TrimSpace(line))
		if len(l) > 30 &&
			!strings.HasPrefix(l, "{") &&
			!strings.HasPrefix(l, "function") &&
			!strings.HasPrefix(l, "window.") &&
			!strings.HasPrefix(l, ".unAuth") &&
			!strings.Contains(l, "gtm.js") &&
			!strings.Contains(l, "backdrop") &&
			!strings.Contains(l, "DrupalUserId") &&
			!seen[l] {
			seen[l] = true
			validLines = append(validLines, l)
		}
	}

	if len(validLines) == 0 {
		return "", applyLink, detailDate
	}

	fullDesc := strings.Join(validLines, "\n\n")
	return fullDesc, applyLink, detailDate
}

const builtInExtractJS = `
(function() {
  const jobs = [];
  const seen = new Set();

  const cards = Array.from(document.querySelectorAll('[data-id="job-card"], .job-bounded-responsive'));

  cards.forEach(card => {
    const titleEl = card.querySelector('[data-id="job-card-title"], a[href*="/job/"]');
    const title = (titleEl?.textContent || titleEl?.getAttribute('title') || '').trim();

    const companyEl = card.querySelector('[data-id="company-title"], a[href*="/company/"]');
    const company = (companyEl?.textContent || '').trim();

    const linkEl = card.querySelector('a[href*="/job/"]');
    let sourceURL = linkEl?.href || '';
    if (sourceURL && !sourceURL.startsWith('http')) {
      sourceURL = 'https://www.builtin.com' + sourceURL;
    }

    const descEl = card.querySelector('.fs-sm, .collapse');
    let desc = (descEl?.textContent || '').trim();
    if (!desc) {
      desc = title + ' at ' + company;
    }

    const locEl = card.querySelector('.font-barlow');
    const location = (locEl?.textContent || 'Remote').trim();

    const clockIcon = card.querySelector('.fa-clock');
    const dateEl = clockIcon ? clockIcon.parentElement : card.querySelector('[title*="Posted"], [title*="reposted"], .text-gray-03');
    const dateStr = (dateEl?.textContent || '').trim();

    const key = sourceURL || (title + '|' + company);
    if (title && title.length > 3 && !seen.has(key)) {
      seen.add(key);
      jobs.push({ title, company: company || 'BuiltIn Company', location, source_url: sourceURL, salary_range: '', posted_at: dateStr, description: desc });
    }
  });

  return JSON.stringify(jobs);
})()
`

func parseBuiltInJobs(jobsJSON, fallbackURL string) ([]models.Job, error) {
	var extracted []struct {
		Title       string `json:"title"`
		Company     string `json:"company"`
		Location    string `json:"location"`
		SourceURL   string `json:"source_url"`
		SalaryRange string `json:"salary_range"`
		PostedAt    string `json:"posted_at"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(jobsJSON), &extracted); err != nil {
		return nil, fmt.Errorf("builtin: parse jobs JSON: %w", err)
	}

	var jobs []models.Job
	for _, e := range extracted {
		url := e.SourceURL
		if url == "" {
			url = fallbackURL
		}

		location := e.Location
		if location == "" {
			location = "Remote"
		}

		job := &models.Job{
			Title:       e.Title,
			Company:     e.Company,
			Location:    location,
			Country:     inferCountry(location),
			SourceURL:   url,
			SourceBoard: "builtin",
			SalaryRange: e.SalaryRange,
			Description: e.Description,
			PostedAt:    parseRelativeDate(e.PostedAt),
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] BuiltIn (chromedp): scraped %d jobs", len(jobs))
	return jobs, nil
}
