package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"time"
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
	if targetURL == "" {
		targetURL = "https://remoteok.com/api"
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

		location := item.Location
		if location == "" {
			location = "Remote"
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
			Country:     inferCountry(location),
			PostedAt:    postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] RemoteOK: fetched %d jobs", len(jobs))
	return jobs, nil
}
