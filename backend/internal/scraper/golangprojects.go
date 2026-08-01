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

// GolangProjectsScraper scrapes https://www.golangprojects.com/golang-remote-jobs.html
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
	resp, err := s.client.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("golangprojects fetch: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("golangprojects parse html: %w", err)
	}

	var jobs []models.Job
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "job-") {
					job := extractGolangProjectsJob(n)
					if job != nil {
						NormalizeJob(job)
						jobs = append(jobs, *job)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	log.Printf("[Scraper] GolangProjects: scraped %d jobs", len(jobs))
	return jobs, nil
}

func extractGolangProjectsJob(n *html.Node) *models.Job {
	job := &models.Job{
		SourceBoard: "golangprojects",
		Country:     "Worldwide",
		Location:    "Remote",
	}

	var getText func(*html.Node) string
	getText = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var result string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			result += getText(c)
		}
		return result
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h2", "h3":
				text := strings.TrimSpace(getText(n))
				if text != "" && job.Title == "" {
					job.Title = text
				}
			case "a":
				for _, a := range n.Attr {
					if a.Key == "href" && strings.Contains(a.Val, "/golang-go-job-") {
						if strings.HasPrefix(a.Val, "http") {
							job.SourceURL = a.Val
						} else {
							job.SourceURL = "https://www.golangprojects.com" + a.Val
						}
					}
				}
				text := strings.TrimSpace(getText(n))
				if text != "" && job.Company == "" && job.Title != "" {
					job.Company = text
				}
			case "p":
				text := strings.TrimSpace(getText(n))
				if text != "" && job.Description == "" && len(text) > 20 {
					job.Description = text
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	if job.Title == "" || job.Company == "" {
		return nil
	}
	if job.Description == "" {
		job.Description = job.Title + " at " + job.Company
	}
	return job
}
