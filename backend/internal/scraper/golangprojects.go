package scraper

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"strings"
	"time"
)

// GolangProjectsScraper parses the GolangProjects RSS feed
type GolangProjectsScraper struct {
	client *http.Client
}

func NewGolangProjectsScraper() *GolangProjectsScraper {
	return &GolangProjectsScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *GolangProjectsScraper) Name() string { return "golangprojects" }

func (s *GolangProjectsScraper) Scrape(targetURL string) ([]models.Job, error) {
	rssURL := "https://www.golangprojects.com/rss.xml"

	req, err := http.NewRequest("GET", rssURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("golangprojects rss fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("golangprojects: HTTP %d", resp.StatusCode)
	}

	type Item struct {
		GUID        string `xml:"guid"`
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
	}

	type RSS struct {
		Items []Item `xml:"channel>item"`
	}

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, fmt.Errorf("golangprojects xml decode: %w", err)
	}

	var jobs []models.Job
	for _, item := range rss.Items {
		// Title format: "Software Engineer (Go) @ Company"
		title := item.Title
		company := "Golang Company"

		if idx := strings.Index(title, " @ "); idx > 0 {
			company = strings.TrimSpace(title[idx+3:])
			title = strings.TrimSpace(title[:idx])
		}

		desc := stripHTML(item.Description)
		if desc == "" {
			desc = title + " at " + company
		}
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}

		link := item.Link
		if link == "" {
			link = item.GUID
		}

		job := &models.Job{
			Title:       title,
			Company:     company,
			SourceURL:   link,
			SourceBoard: "golangprojects",
			Description: desc,
			Location:    "Remote",
			Country:     inferCountry(desc),
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] GolangProjects: scraped %d jobs", len(jobs))
	return jobs, nil
}
