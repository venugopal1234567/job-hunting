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

// VacancyGlobalProScraper scrapes jobs from VacancyGlobalPro
type VacancyGlobalProScraper struct {
	client *http.Client
}

func NewVacancyGlobalProScraper() *VacancyGlobalProScraper {
	return &VacancyGlobalProScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *VacancyGlobalProScraper) Name() string { return "vacancyglobalpro" }

func (s *VacancyGlobalProScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://vacancyglobalpro.up.railway.app/remote-golang-jobs"
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
		return nil, fmt.Errorf("vacancyglobalpro fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vacancyglobalpro: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlStr := string(body)
	parts := strings.Split(htmlStr, `href="/job/`)

	reTitle := regexp.MustCompile(`(?s)class="job-title">([^<]+)</a>`)
	reCompany := regexp.MustCompile(`(?s)<div class="job-company">([^<]+)</div>`)

	var jobs []models.Job
	seen := make(map[string]bool)

	for _, part := range parts[1:] {
		// Extract job path (e.g. senior-golang-engineer-global-loyalty-benefits" class="job-title")
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

		companyMatch := reCompany.FindStringSubmatch(part)
		company := "VacancyGlobalPro"
		if len(companyMatch) >= 2 {
			company = strings.TrimSpace(companyMatch[1])
		}

		job := models.Job{
			Title:       title,
			Company:     company,
			Location:    "Remote",
			Country:     "Worldwide",
			SourceURL:   detailURL, // fallback
			SourceBoard: "vacancyglobalpro",
			Description: title + " position",
		}
		jobs = append(jobs, job)
	}

	// Fetch detail page concurrently
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	var mu sync.Mutex
	var finalJobs []models.Job

	for i := range jobs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			j := jobs[idx]
			s.fetchDetail(&j)
			NormalizeJob(&j)

			// Skip US jobs immediately
			c := strings.ToLower(j.Country)
			l := strings.ToLower(j.Location)
			if c == "us" || c == "usa" || strings.Contains(c, "united states") || strings.Contains(c, "us only") ||
				l == "us" || l == "usa" || strings.Contains(l, "united states") || strings.Contains(l, "us only") {
				return
			}

			mu.Lock()
			finalJobs = append(finalJobs, j)
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	log.Printf("[Scraper] VacancyGlobalPro: scraped %d jobs (skipped US/USA)", len(finalJobs))
	return finalJobs, nil
}

type vacancyJobLD struct {
	Type        string `json:"@type"`
	Title       string `json:"title"`
	DatePosted  string `json:"datePosted"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

func (s *VacancyGlobalProScraper) fetchDetail(job *models.Job) {
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

	// Extract location from meta item containing map-pin path
	reMeta := regexp.MustCompile(`(?s)<div class="job-detail-meta-item">([\s\S]*?)</div>`)
	metaMatches := reMeta.FindAllStringSubmatch(htmlStr, -1)
	for _, mm := range metaMatches {
		content := mm[1]
		if strings.Contains(content, "M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z") {
			locText := stripHTML(content)
			job.Location = strings.TrimSpace(locText)
			job.Country = inferCountry(job.Location)
			break
		}
	}

	// Try application/ld+json
	reJSONLD := regexp.MustCompile(`(?s)<script type="application/ld\+json">([\s\S]*?)</script>`)
	matches := reJSONLD.FindAllStringSubmatch(htmlStr, -1)

	var ld vacancyJobLD
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
		if ld.DatePosted != "" {
			if t, err := time.Parse("2006-01-02", ld.DatePosted); err == nil {
				job.PostedAt = &t
			}
		}
	} else {
		// Fallback description
		reDesc := regexp.MustCompile(`(?s)<div class="job-content">([\s\S]*?)</div>`)
		descMatch := reDesc.FindStringSubmatch(htmlStr)
		if len(descMatch) >= 2 {
			desc := stripHTML(descMatch[1])
			if len(desc) > 3000 {
				desc = desc[:3000] + "..."
			}
			job.Description = desc
		}

		reApply := regexp.MustCompile(`(?s)<a href="([^"]+)"\s+class="apply-button-main"`)
		applyMatch := reApply.FindStringSubmatch(htmlStr)
		if len(applyMatch) >= 2 {
			job.SourceURL = applyMatch[1]
		}
	}
}
