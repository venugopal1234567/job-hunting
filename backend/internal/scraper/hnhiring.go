package scraper

import (
	"fmt"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// HNHiringScraper scrapes https://hnhiring.com/technologies/go
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
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RemoteHunter/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hnhiring fetch: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hnhiring parse html: %w", err)
	}

	var jobs []models.Job
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			job := extractHNJob(n)
			if job != nil {
				NormalizeJob(job)
				jobs = append(jobs, *job)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	log.Printf("[Scraper] HNHiring: scraped %d jobs", len(jobs))
	return jobs, nil
}

func extractHNJob(n *html.Node) *models.Job {
	job := &models.Job{
		SourceBoard: "hnhiring",
		Location:    "Remote",
		Country:     "Worldwide",
		SourceURL:   "https://hnhiring.com/technologies/go",
	}

	var getText func(*html.Node) string
	getText = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var sb strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(getText(c))
		}
		return sb.String()
	}

	fullText := strings.TrimSpace(getText(n))
	lines := strings.SplitN(fullText, "\n", 5)

	if len(lines) >= 1 {
		job.Company = strings.TrimSpace(lines[0])
	}
	if len(lines) >= 2 {
		job.Title = strings.TrimSpace(lines[1])
	}
	if job.Title == "" {
		job.Title = "Go Engineer"
	}
	if len(lines) >= 3 {
		job.Description = strings.Join(lines[2:], " ")
	}
	if job.Description == "" {
		job.Description = fullText
	}

	// Try to extract URL from anchor
	var walkLinks func(*html.Node)
	walkLinks = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && (strings.HasPrefix(a.Val, "http") || strings.HasPrefix(a.Val, "/")) {
					if !strings.Contains(a.Val, "hnhiring.com") {
						job.SourceURL = a.Val
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkLinks(c)
		}
	}
	walkLinks(n)

	if job.Company == "" {
		return nil
	}
	return job
}
