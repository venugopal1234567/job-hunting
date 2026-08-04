package scraper

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"remotehunter/internal/models"
	"strings"
	"sync"
	"time"
)

// GolangProjectsScraper parses the GolangProjects RSS feed and detail pages
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
		title := item.Title
		company := "Golang Company"

		// Title format can contain non-breaking spaces (\u00a0)
		title = strings.ReplaceAll(title, "\u00a0", " ")
		title = strings.ReplaceAll(title, "&nbsp;", " ")

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
		jobs = append(jobs, *job)
	}

	// Concurrently fetch detail page to get correct PostedAt date and full description
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for i := range jobs {
		if jobs[i].SourceURL == "" || !strings.HasPrefix(jobs[i].SourceURL, "http") {
			NormalizeJob(&jobs[i])
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			posted, fullDesc := s.fetchDetailInfo(jobs[idx].SourceURL)
			if posted != nil {
				jobs[idx].PostedAt = posted
			}
			if len(fullDesc) > 50 {
				jobs[idx].Description = fullDesc
			}
			NormalizeJob(&jobs[idx])
		}(i)
	}
	wg.Wait()

	log.Printf("[Scraper] GolangProjects: scraped %d jobs", len(jobs))
	return jobs, nil
}

func (s *GolangProjectsScraper) fetchDetailInfo(detailURL string) (*time.Time, string) {
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil, ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ""
	}

	htmlStr := string(body)

	// 1. Extract date Posted
	var postedDate *time.Time
	reDate := regexp.MustCompile(`"datePosted"\s*:\s*"(\d{4}-\d{2}-\d{2})"`)
	if m := reDate.FindStringSubmatch(htmlStr); len(m) >= 2 {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			postedDate = &t
		}
	}

	// 2. Extract description (JSON-LD schema "description")
	var description string
	reDesc := regexp.MustCompile(`"description"\s*:\s*"([^"]+)"`)
	if m := reDesc.FindStringSubmatch(htmlStr); len(m) >= 2 {
		// Clean up escape characters
		descClean := strings.ReplaceAll(m[1], `\n`, "\n")
		descClean = strings.ReplaceAll(descClean, `\r`, "")
		descClean = strings.ReplaceAll(descClean, `\"`, "\"")
		descClean = strings.ReplaceAll(descClean, `\t`, " ")
		description = stripHTML(descClean)
	}

	return postedDate, description
}
