package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"remotehunter/internal/models"
	"strings"
	"time"
)

type IndeedScraper struct {
	client *http.Client
}

func NewIndeedScraper() *IndeedScraper {
	return &IndeedScraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *IndeedScraper) Name() string { return "indeed" }

func (s *IndeedScraper) Scrape(targetURL string) ([]models.Job, error) {
	// Query string extraction. If empty or generic, use defaults.
	searchTerm := "golang"
	if targetURL != "" && strings.Contains(targetURL, "what=") {
		parts := strings.Split(targetURL, "what=")
		if len(parts) > 1 {
			searchTerm = strings.Split(parts[1], "&")[0]
		}
	}

	apiURL := "https://apis.indeed.com/graphql"
	query := fmt.Sprintf(`query GetJobData {
		jobSearch(
			what: "%s",
			limit: 100,
			filters: {
				composite: {
					filters: [{
						keyword: {
							field: "attributes",
							keys: ["DSQF7"]
						}
					}]
				}
			}
		) {
			results {
				job {
					key
					title
					datePublished
					description {
						html
					}
					employer {
						name
						relativeCompanyPageUrl
					}
					location {
						countryName
						formatted {
							short
						}
					}
				}
			}
		}
	}`, searchTerm)

	payload := map[string]string{
		"query": query,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Host", "apis.indeed.com")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("indeed-api-key", "161092c2017b5bbab13edb12461a62d5a833871e7cad6d9d475304573de67ac8")
	req.Header.Set("accept", "application/json")
	req.Header.Set("indeed-locale", "en-US")
	req.Header.Set("indeed-co", "IN") // default to India scope or US scope
	req.Header.Set("user-agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Indeed App 193.1")
	req.Header.Set("indeed-app-info", "appv=193.1; appid=com.indeed.jobsearch; osv=16.6.1; os=ios; dtype=phone")

	log.Printf("[Scraper] Indeed: Fetching jobs via GraphQL...")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("indeed: API returned status %d: %s", resp.StatusCode, string(body))
	}

	type GraphQLResponse struct {
		Data struct {
			JobSearch struct {
				Results []struct {
					Job struct {
						Key           string `json:"key"`
						Title         string `json:"title"`
						DatePublished int64  `json:"datePublished"`
						Description   struct {
							HTML string `json:"html"`
						} `json:"description"`
						Employer *struct {
							Name                   string `json:"name"`
							RelativeCompanyPageURL string `json:"relativeCompanyPageUrl"`
						} `json:"employer"`
						Location struct {
							CountryName string `json:"countryName"`
							Formatted   struct {
								Short string `json:"short"`
							} `json:"formatted"`
						} `json:"location"`
					} `json:"job"`
				} `json:"results"`
			} `json:"jobSearch"`
		} `json:"data"`
	}

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, err
	}

	var jobs []models.Job
	seen := make(map[string]bool)

	for _, res := range gqlResp.Data.JobSearch.Results {
		rj := res.Job
		if rj.Key == "" || rj.Title == "" {
			continue
		}

		compName := "N/A"
		if rj.Employer != nil {
			compName = rj.Employer.Name
		}

		location := rj.Location.Formatted.Short
		country := "Worldwide"
		if rj.Location.CountryName != "" {
			country = rj.Location.CountryName
		}
		if country == "IN" || country == "India" {
			country = "India"
		}

		desc := rj.Description.HTML
		// Minimal tag stripping for markdown format
		desc = strings.ReplaceAll(desc, "<p>", "")
		desc = strings.ReplaceAll(desc, "</p>", "\n")
		desc = strings.ReplaceAll(desc, "<br>", "\n")
		desc = strings.ReplaceAll(desc, "<br/>", "\n")
		desc = strings.ReplaceAll(desc, "<li>", "- ")
		desc = strings.ReplaceAll(desc, "</li>", "\n")
		// Remove remaining html tags
		for {
			start := strings.Index(desc, "<")
			if start == -1 {
				break
			}
			end := strings.Index(desc[start:], ">")
			if end == -1 {
				break
			}
			desc = desc[:start] + desc[start+end+1:]
		}

		desc = strings.TrimSpace(desc)
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}
		if desc == "" {
			desc = rj.Title + " at " + compName
		}

		postedAt := time.Now()
		if rj.DatePublished > 0 {
			postedAt = time.Unix(rj.DatePublished/1000, 0)
		}

		job := &models.Job{
			Title:       rj.Title,
			Company:     compName,
			SourceURL:   fmt.Sprintf("https://www.indeed.com/viewjob?jk=%s", rj.Key),
			SourceBoard: "indeed",
			Description: desc,
			Location:    location,
			Country:     country,
			JobType:     "Full Time",
			PostedAt:    &postedAt,
		}

		NormalizeJob(job)
		if !seen[job.JobHash] {
			seen[job.JobHash] = true
			jobs = append(jobs, *job)
		}
	}

	log.Printf("[Scraper] Indeed: scraped %d jobs", len(jobs))
	return jobs, nil
}
