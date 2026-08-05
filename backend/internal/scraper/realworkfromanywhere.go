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

type RealWorkFromAnywhereScraper struct{}

func NewRealWorkFromAnywhereScraper() *RealWorkFromAnywhereScraper {
	return &RealWorkFromAnywhereScraper{}
}

func (s *RealWorkFromAnywhereScraper) Name() string { return "realworkfromanywhere" }

func (s *RealWorkFromAnywhereScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://www.realworkfromanywhere.com/remote-backend-jobs"
	}

	urls := strings.Split(targetURL, "|")
	var allJobs []models.Job
	seen := make(map[string]bool)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		log.Printf("[Scraper] RealWorkFromAnywhere: Fetching %s via HTTP GET", u)
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			log.Printf("[Scraper] RealWorkFromAnywhere: request creation error: %v", err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Scraper] RealWorkFromAnywhere: HTTP request failed: %v", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[Scraper] RealWorkFromAnywhere: received HTTP status %d", resp.StatusCode)
			continue
		}

		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Printf("[Scraper] RealWorkFromAnywhere: HTML parsing failed: %v", err)
			continue
		}

		walkHTML(doc, &allJobs, seen)
	}

	log.Printf("[Scraper] RealWorkFromAnywhere: scraped %d unique Go jobs", len(allJobs))
	return allJobs, nil
}

func findVal(n *html.Node, attr string) string {
	for _, a := range n.Attr {
		if a.Key == attr {
			return a.Val
		}
	}
	return ""
}

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(getText(c))
	}
	return b.String()
}

func walkHTML(n *html.Node, jobs *[]models.Job, seen map[string]bool) {
	if n.Type == html.ElementNode && n.Data == "a" {
		href := findVal(n, "href")
		if strings.Contains(href, "/jobs/") {
			var title, company, location string
			var badges []string
			var dateText string

			var parseAnchor func(*html.Node)
			parseAnchor = func(child *html.Node) {
				if child.Type == html.ElementNode {
					if child.Data == "h3" {
						title = strings.TrimSpace(getText(child))
					} else if child.Data == "p" {
						// The company name is usually a p tag inside the anchor
						company = strings.TrimSpace(getText(child))
					} else if child.Data == "span" {
						txt := strings.TrimSpace(getText(child))
						if strings.Contains(txt, "ago") {
							dateText = txt
						} else {
							class := findVal(child, "class")
							if strings.Contains(class, "badge") {
								badges = append(badges, txt)
							}
						}
					} else if child.Data == "div" {
						class := findVal(child, "class")
						if strings.Contains(class, "items-center") && strings.Contains(class, "gap-1.5") {
							// Extract location text (excluding svg elements)
							location = strings.TrimSpace(getText(child))
						}
					}
				}
				for c := child.FirstChild; c != nil; c = c.NextSibling {
					parseAnchor(c)
				}
			}
			parseAnchor(n)

			if title != "" && company != "" {
				titleLower := strings.ToLower(title)
				hasGo := false
				words := strings.FieldsFunc(titleLower, func(c rune) bool {
					return c == ' ' || c == '/' || c == '-' || c == ',' || c == '(' || c == ')' || c == '[' || c == ']'
				})
				for _, w := range words {
					if w == "go" || w == "golang" {
						hasGo = true
						break
					}
				}
				for _, badge := range badges {
					b := strings.ToLower(badge)
					if b == "go" || b == "golang" || b == "go lang" {
						hasGo = true
						break
					}
				}

				if hasGo {
					jobURL := href
					if !strings.HasPrefix(jobURL, "http") {
						jobURL = "https://www.realworkfromanywhere.com" + jobURL
					}

					country := "Worldwide"
					if strings.Contains(strings.ToLower(location), "india") {
						country = "India"
					} else if strings.Contains(strings.ToLower(location), "us") || strings.Contains(strings.ToLower(location), "united states") {
						country = "US"
					}

					desc := title + " at " + company + " (Location: " + location + ")\nTags: " + strings.Join(badges, ", ")
					postedAt := parseRealWFADate(dateText)

					job := &models.Job{
						Title:       title,
						Company:     company,
						SourceURL:   jobURL,
						SourceBoard: "realworkfromanywhere",
						Description: desc,
						Location:    location,
						Country:     country,
						JobType:     "Full Time",
						PostedAt:    postedAt,
					}
					NormalizeJob(job)
					if !seen[job.JobHash] {
						seen[job.JobHash] = true
						*jobs = append(*jobs, *job)
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkHTML(c, jobs, seen)
	}
}

func parseRealWFADate(dateText string) *time.Time {
	dateText = strings.ToLower(dateText)
	now := time.Now()

	if strings.Contains(dateText, "today") || strings.Contains(dateText, "just now") || strings.Contains(dateText, "hour") {
		return &now
	}

	if strings.Contains(dateText, "yesterday") || strings.Contains(dateText, "1 day ago") {
		t := now.AddDate(0, 0, -1)
		return &t
	}

	var days int
	if strings.Contains(dateText, "week") {
		days = 7
		if strings.Contains(dateText, "2") {
			days = 14
		} else if strings.Contains(dateText, "3") {
			days = 21
		} else if strings.Contains(dateText, "4") {
			days = 28
		}
	} else if strings.Contains(dateText, "month") {
		days = 30
	} else {
		for d := 2; d <= 30; d++ {
			if strings.Contains(dateText, fmt.Sprintf("%d day", d)) {
				days = d
				break
			}
		}
	}

	if days > 0 {
		t := now.AddDate(0, 0, -days)
		return &t
	}

	return nil
}
