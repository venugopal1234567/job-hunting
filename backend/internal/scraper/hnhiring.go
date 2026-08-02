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

// HNHiringScraper uses Hacker News Algolia API to fetch real-time remote developer postings
type HNHiringScraper struct {
	client *http.Client
}

func NewHNHiringScraper() *HNHiringScraper {
	return &HNHiringScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *HNHiringScraper) Name() string { return "hnhiring" }

func (s *HNHiringScraper) Scrape(targetURL string) ([]models.Job, error) {
	apiURL := "https://hn.algolia.com/api/v1/search?query=golang+remote&tags=comment&hitsPerPage=40"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hnhiring fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hnhiring: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Hits []struct {
			ObjectID    string `json:"objectID"`
			CommentText string `json:"comment_text"`
			StoryID     int    `json:"story_id"`
			CreatedAt   string `json:"created_at"`
			Author      string `json:"author"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("hnhiring decode: %w", err)
	}

	var jobs []models.Job
	for _, hit := range result.Hits {
		text := stripHTML(hit.CommentText)
		if len(text) < 30 {
			continue
		}

		lines := strings.Split(text, "|")
		company := "HN Startup"
		title := "Go / Backend Engineer"

		if len(lines) > 0 {
			company = strings.TrimSpace(lines[0])
		}
		if len(lines) > 1 {
			title = strings.TrimSpace(lines[1])
		}

		if company == "" {
			company = "HackerNews Poster"
		}
		if title == "" {
			title = "Software Engineer (Golang / Remote)"
		}

		desc := text
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}

		url := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)

		var postedAt *time.Time
		if hit.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, hit.CreatedAt); err == nil {
				postedAt = &t
			}
		}

		job := &models.Job{
			Title:       title,
			Company:     company,
			SourceURL:   url,
			SourceBoard: "hnhiring",
			Description: desc,
			Location:    "Remote",
			Country:     inferCountry(desc),
			PostedAt:    postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] HNHiring: fetched %d jobs", len(jobs))
	return jobs, nil
}
