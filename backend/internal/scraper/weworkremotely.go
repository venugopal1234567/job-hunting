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

// WeWorkRemotelyScraper scrapes WeWorkRemotely via their programming jobs RSS feed
type WeWorkRemotelyScraper struct {
	client *http.Client
}

func NewWeWorkRemotelyScraper() *WeWorkRemotelyScraper {
	return &WeWorkRemotelyScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
			// Follow redirects
		},
	}
}

func (s *WeWorkRemotelyScraper) Name() string { return "weworkremotely" }

func (s *WeWorkRemotelyScraper) Scrape(targetURL string) ([]models.Job, error) {
	// Use the confirmed-working programming jobs RSS feed
	// The /remote-jobs/search URL returns HTML — the RSS feed is the reliable endpoint
	rssURL := "https://weworkremotely.com/categories/remote-programming-jobs.rss"

	req, err := http.NewRequest("GET", rssURL, nil)
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
		// CDATA wrapped text
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
		// WWR title format: "[Company]: Job Title at Location"
		// e.g. "Stripe: Senior Backend Engineer at Remote"
		title := item.Title
		company := "Unknown"

		if idx := strings.Index(title, ": "); idx > 0 {
			company = strings.TrimSpace(title[:idx])
			title = strings.TrimSpace(title[idx+2:])
		}

		// Strip "at <location>" suffix from title
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

		// Parse date
		if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
			job.PostedAt = &t
		} else if t, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
			job.PostedAt = &t
		}

		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] WeWorkRemotely: scraped %d jobs", len(jobs))
	return jobs, nil
}

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
	// Collapse whitespace
	return strings.Join(strings.Fields(result.String()), " ")
}
