package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"remotehunter/internal/models"
	"strings"
	"time"
)

var (
	goWordRegex       = regexp.MustCompile(`(?i)\b(go|golang)\b`)
	excludeTitleRegex = regexp.MustCompile(`(?i)\b(go-to-market|go to market|gtm|sales|marketing|sdr|bdr|representative|recruiter|talent|business\s+development|estimator|technician|cleaning|cleaner|artist|writer|content|community|customer\s+support|success|support\s+engineer|operation)\b`)
	techTitleRegex    = regexp.MustCompile(`(?i)\b(backend|software|engineer|developer|programmer|architect|fullstack|full-stack|tech\s+lead|systems|lead|senior|junior|staff|principal|coder)\b`)
)

// RemoteOKScraper scrapes jobs from RemoteOK public JSON API
type RemoteOKScraper struct {
	client *http.Client
}

func NewRemoteOKScraper() *RemoteOKScraper {
	return &RemoteOKScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *RemoteOKScraper) Name() string { return "remoteok" }

func (s *RemoteOKScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" || !strings.Contains(targetURL, "tag=") {
		targetURL = "https://remoteok.com/api?tag=golang"
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remoteok fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remoteok: HTTP %d", resp.StatusCode)
	}

	var rawItems []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&rawItems); err != nil {
		return nil, fmt.Errorf("remoteok decode: %w", err)
	}

	var jobs []models.Job
	for _, raw := range rawItems {
		var item struct {
			ID          string   `json:"id"`
			Slug        string   `json:"slug"`
			Company     string   `json:"company"`
			Position    string   `json:"position"`
			Description string   `json:"description"`
			Location    string   `json:"location"`
			URL         string   `json:"url"`
			Tags        []string `json:"tags"`
			Epoch       int64    `json:"epoch"`
		}

		if err := json.Unmarshal(raw, &item); err != nil || item.Position == "" || item.Company == "" {
			continue // Skip legal notice item or invalid elements
		}

		// 1. Verify if the job is truly Golang related (avoid tag spam)
		if !isGolangJob(item.Position, item.Description, item.Tags) {
			continue
		}

		// 2. Normalize and check location (must be Worldwide or India)
		location := item.Location
		if location == "" {
			location = "Remote"
		}
		country := inferCountryRemoteOK(location)
		if country != "Worldwide" && country != "India" {
			continue
		}

		url := item.URL
		if url == "" && item.Slug != "" {
			url = "https://remoteok.com/remote-jobs/" + item.Slug
		}

		var postedAt *time.Time
		if item.Epoch > 0 {
			t := time.Unix(item.Epoch, 0)
			postedAt = &t
		}

		desc := stripHTML(item.Description)
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}
		if desc == "" {
			desc = item.Position + " at " + item.Company
		}

		job := &models.Job{
			Title:       item.Position,
			Company:     item.Company,
			SourceURL:   url,
			SourceBoard: "remoteok",
			Description: desc,
			Location:    location,
			Country:     country,
			PostedAt:    postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] RemoteOK: fetched %d filtered jobs", len(jobs))
	return jobs, nil
}
func isGolangJob(title, description string, tags []string) bool {
	// If it's a non-dev/sales/other excluded role, immediately reject
	if excludeTitleRegex.MatchString(title) {
		return false
	}

	// If title contains "go" or "golang" as a word, it's definitely Go
	if goWordRegex.MatchString(title) {
		return true
	}

	// Check tags for "go" or "golang"
	hasGoTag := false
	for _, tag := range tags {
		t := strings.ToLower(tag)
		if t == "golang" || t == "go" {
			hasGoTag = true
			break
		}
	}

	// If it has a tag, make sure it's a technical job and tag list is not spammed (<= 10 tags)
	if hasGoTag && len(tags) <= 10 && techTitleRegex.MatchString(title) {
		return true
	}

	return false
}

func inferCountryRemoteOK(location string) string {
	loc := strings.ToLower(location)
	if loc == "" || strings.Contains(loc, "worldwide") || strings.Contains(loc, "anywhere") || loc == "remote" {
		return "Worldwide"
	}

	// Precise India matching to prevent "indiana" / "indonesia" false positives
	if strings.Contains(loc, "india") || strings.Contains(loc, "in") {
		if strings.Contains(loc, "indiana") || strings.Contains(loc, "indonesia") || strings.Contains(loc, "indies") {
			// Skip/do nothing
		} else {
			return "India"
		}
	}

	cities := []string{"bangalore", "bengaluru", "karnataka", "pune", "hyderabad", "chennai", "mumbai", "delhi", "noida", "gurgaon"}
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

