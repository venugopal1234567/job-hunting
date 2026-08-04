package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"strings"
	"time"
)

// RemotiveScraper uses the Remotive.com public JSON API
// Docs: https://remotive.com/api/remote-jobs
type RemotiveScraper struct {
	client *http.Client
}

func NewRemotiveScraper() *RemotiveScraper {
	return &RemotiveScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *RemotiveScraper) Name() string { return "remotive" }

func (s *RemotiveScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" || strings.Contains(targetURL, "search=") {
		targetURL = "https://remotive.com/api/remote-jobs?category=software-dev&limit=50"
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remotive fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remotive: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Jobs []struct {
			ID                       int      `json:"id"`
			URL                      string   `json:"url"`
			Title                    string   `json:"title"`
			CompanyName              string   `json:"company_name"`
			Category                 string   `json:"category"`
			Tags                     []string `json:"tags"`
			JobType                  string   `json:"job_type"`
			PublicationDate          string   `json:"publication_date"`
			CandidateRequiredLocation string  `json:"candidate_required_location"`
			Salary                   string   `json:"salary"`
			Description              string   `json:"description"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("remotive decode: %w", err)
	}

	var jobs []models.Job
	for _, item := range result.Jobs {
		location := item.CandidateRequiredLocation
		if location == "" {
			location = "Remote"
		}
		country := inferCountry(location)

		// Parse date
		var postedAt *time.Time
		if item.PublicationDate != "" {
			if t, err := time.Parse("2006-01-02T15:04:05", item.PublicationDate); err == nil {
				postedAt = &t
			} else if t, err := time.Parse("2006-01-02", item.PublicationDate[:10]); err == nil {
				postedAt = &t
			}
		}

		desc := stripHTML(item.Description)
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}

		job := &models.Job{
			Title:       item.Title,
			Company:     item.CompanyName,
			SourceURL:   item.URL,
			SourceBoard: "remotive",
			Description: desc,
			SalaryRange: item.Salary,
			JobType:     item.JobType,
			Location:    location,
			Country:     country,
			PostedAt:    postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] Remotive: fetched %d jobs", len(jobs))
	return jobs, nil
}

// inferCountry extracts a country code from a location string
func inferCountry(location string) string {
	loc := strings.ToLower(location)
	switch {
	case strings.Contains(loc, "worldwide") || strings.Contains(loc, "anywhere") || loc == "remote":
		return "Worldwide"
	case strings.Contains(loc, "usa") || strings.Contains(loc, "united states") || strings.Contains(loc, "us only"):
		return "US"
	case strings.Contains(loc, "europe") || strings.Contains(loc, "eu only"):
		return "Europe"
	case strings.Contains(loc, "india") || strings.Contains(loc, "ind") || strings.Contains(loc, "karnataka") || strings.Contains(loc, "bangalore") || strings.Contains(loc, "bengaluru"):
		return "India"
	case strings.Contains(loc, "uk") || strings.Contains(loc, "united kingdom"):
		return "UK"
	case strings.Contains(loc, "remote"):
		return "Worldwide"
	default:
		return location
	}
}
