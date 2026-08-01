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

// ArbeitnowScraper uses the free Arbeitnow job board API
// Docs: https://www.arbeitnow.com/api/job-board-api
type ArbeitnowScraper struct {
	client *http.Client
}

func NewArbeitnowScraper() *ArbeitnowScraper {
	return &ArbeitnowScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *ArbeitnowScraper) Name() string { return "arbeitnow" }

func (s *ArbeitnowScraper) Scrape(targetURL string) ([]models.Job, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RemoteHunter/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arbeitnow fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arbeitnow: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Slug        string   `json:"slug"`
			CompanyName string   `json:"company_name"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Remote      bool     `json:"remote"`
			URL         string   `json:"url"`
			Tags        []string `json:"tags"`
			JobTypes    []string `json:"job_types"`
			Location    string   `json:"location"`
			CreatedAt   int64    `json:"created_at"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("arbeitnow decode: %w", err)
	}

	var jobs []models.Job
	for _, item := range result.Data {
		if !item.Remote {
			continue // only remote jobs
		}

		location := item.Location
		if location == "" {
			location = "Remote"
		}

		postedAt := time.Unix(item.CreatedAt, 0)
		jobType := ""
		if len(item.JobTypes) > 0 {
			jobType = strings.Join(item.JobTypes, ", ")
		}

		// Use slug-based URL if canonical URL is missing
		sourceURL := item.URL
		if sourceURL == "" {
			sourceURL = "https://www.arbeitnow.com/jobs/" + item.Slug
		}

		desc := stripHTML(item.Description)
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}
		if desc == "" {
			desc = item.Title + " at " + item.CompanyName
		}

		job := &models.Job{
			Title:       item.Title,
			Company:     item.CompanyName,
			SourceURL:   sourceURL,
			SourceBoard: "arbeitnow",
			Description: desc,
			JobType:     jobType,
			Location:    location,
			Country:     inferCountry(location),
			PostedAt:    &postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] Arbeitnow: fetched %d remote jobs", len(jobs))
	return jobs, nil
}
