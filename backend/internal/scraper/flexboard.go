package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"remotehunter/internal/models"
	"strings"
	"sync"
	"time"
)

// FlexboardScraper scrapes jobs from Flexboard
type FlexboardScraper struct {
	client *http.Client
}

func NewFlexboardScraper() *FlexboardScraper {
	return &FlexboardScraper{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *FlexboardScraper) Name() string { return "flexboard" }

func (s *FlexboardScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://flexboard.9y.liveblog365.com/?search=golang"
	}

	parsedBase, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target url: %w", err)
	}
	baseURL := parsedBase.Scheme + "://" + parsedBase.Host

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flexboard fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("flexboard: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlStr := string(body)
	parts := strings.Split(htmlStr, `href="/job/`)

	reTitle := regexp.MustCompile(`(?s)<h3 class="job-title">([^<]+)</h3>`)
	reMeta := regexp.MustCompile(`(?s)<span>([^<]+)</span>`)

	var jobs []models.Job
	seen := make(map[string]bool)

	for _, part := range parts[1:] {
		// Extract job path (e.g. 2723601" class="job-card")
		idx := strings.Index(part, `"`)
		if idx == -1 {
			continue
		}
		jobID := part[:idx]
		detailURL := baseURL + "/job/" + jobID

		if seen[detailURL] {
			continue
		}
		seen[detailURL] = true

		titleMatch := reTitle.FindStringSubmatch(part)
		title := "Golang Developer"
		if len(titleMatch) >= 2 {
			title = strings.TrimSpace(titleMatch[1])
		}

		// Parse metadata (location, country)
		var metaSpans []string
		metaMatches := reMeta.FindAllStringSubmatch(part, -1)
		for _, m := range metaMatches {
			if len(m) >= 2 {
				metaSpans = append(metaSpans, strings.TrimSpace(m[1]))
			}
		}

		location := "Remote"
		country := "Worldwide"
		if len(metaSpans) >= 1 {
			location = metaSpans[0]
		}
		if len(metaSpans) >= 2 {
			country = metaSpans[1]
		}

		job := models.Job{
			Title:       title,
			Company:     "FlexBoard",
			Location:    location,
			Country:     inferCountry(country),
			SourceURL:   detailURL, // fallback
			SourceBoard: "flexboard",
			Description: title + " position",
		}
		jobs = append(jobs, job)
	}

	// Fetch detail page concurrently for description, actual apply link, and date
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i := range jobs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s.fetchDetail(&jobs[idx])
			NormalizeJob(&jobs[idx])
		}(i)
	}
	wg.Wait()

	log.Printf("[Scraper] Flexboard: scraped %d jobs", len(jobs))
	return jobs, nil
}

type flexboardJobLD struct {
	Type           string `json:"@type"`
	Title          string `json:"title"`
	DatePosted     string `json:"datePosted"`
	Description    string `json:"description"`
	URL            string `json:"url"`
	EmploymentType string `json:"employmentType"`
}

func (s *FlexboardScraper) fetchDetail(job *models.Job) {
	req, err := http.NewRequest("GET", job.SourceURL, nil)
	if err != nil {
		job.PostedAt = nil
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		job.PostedAt = nil
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		job.PostedAt = nil
		return
	}

	htmlStr := string(body)

	// Try application/ld+json first
	reJSONLD := regexp.MustCompile(`(?s)<script type="application/ld\+json">([\s\S]*?)</script>`)
	matches := reJSONLD.FindAllStringSubmatch(htmlStr, -1)

	var ld flexboardJobLD
	foundLD := false

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(m[1]), &parsed); err == nil {
			if parsed["@type"] == "JobPosting" {
				if data, err := json.Marshal(parsed); err == nil {
					if err := json.Unmarshal(data, &ld); err == nil {
						foundLD = true
						break
					}
				}
			}
		}
	}

	if foundLD {
		if ld.Title != "" {
			job.Title = ld.Title
		}
		if ld.URL != "" {
			job.SourceURL = ld.URL
		}
		if ld.Description != "" {
			desc := stripHTML(ld.Description)
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}
			job.Description = desc
		}
	} else {
		// Fallback HTML parsing for description and apply URL
		reDesc := regexp.MustCompile(`(?s)<div class="job-description">([\s\S]*?)</div>`)
		descMatch := reDesc.FindStringSubmatch(htmlStr)
		if len(descMatch) >= 2 {
			desc := stripHTML(descMatch[1])
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}
			job.Description = desc
		}

		reApply := regexp.MustCompile(`(?s)<a href="([^"]+)"\s+class="apply-button"`)
		applyMatch := reApply.FindStringSubmatch(htmlStr)
		if len(applyMatch) >= 2 {
			job.SourceURL = applyMatch[1]
		}
	}

	// Fetch actual post date from original page if possible
	if job.SourceURL != "" && strings.HasPrefix(job.SourceURL, "http") {
		realDate := s.fetchExternalDate(job.SourceURL)
		if realDate != nil {
			job.PostedAt = realDate
			return
		}
	}

	// FlexBoard's datePosted in JSON-LD is hardcoded to today's date, so we set it to nil to avoid incorrect dates
	job.PostedAt = nil
}

func (s *FlexboardScraper) fetchExternalDate(url string) *time.Time {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	htmlStr := string(body)
	reDate := regexp.MustCompile(`(?i)(?:Posted\s+)?(\d+\+?\s*(?:minute|hour|day|week|month|year)s?\s*ago|today|yesterday|just now)`)
	m := reDate.FindStringSubmatch(htmlStr)
	if len(m) >= 2 {
		return parseRelativeDate(m[1])
	}
	return nil
}
