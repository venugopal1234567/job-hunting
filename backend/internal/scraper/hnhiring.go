package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"remotehunter/internal/models"
	"strconv"
	"strings"
	"time"
)

// HNHiringScraper uses the Hacker News Algolia API to pull job postings from the
// monthly "Ask HN: Who is hiring?" thread.
//
// The thread is resolved by ID first: Algolia matches comments across every
// thread authored by `whoishiring`, which includes the sibling "Who wants to be
// hired?" thread (job seekers posting their own résumés). Scraping those
// unfiltered produced rows whose "company" was a candidate's location line.
type HNHiringScraper struct {
	client *http.Client
}

func NewHNHiringScraper() *HNHiringScraper {
	return &HNHiringScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *HNHiringScraper) Name() string { return "hnhiring" }

// goKeyword matches "go" as a standalone word so "Google", "Golang", "Pango"
// and "going" don't pull in unrelated postings.
var goKeyword = regexp.MustCompile(`(?i)\b(go|golang)\b`)

func (s *HNHiringScraper) Scrape(targetURL string) ([]models.Job, error) {
	storyID, err := s.latestHiringStory()
	if err != nil {
		return nil, err
	}
	log.Printf("[Scraper] HNHiring: latest hiring thread story_id=%d", storyID)

	comments, err := s.threadComments(storyID)
	if err != nil {
		return nil, err
	}

	var jobs []models.Job
	for _, hit := range comments {
		// Only top-level comments are job postings; everything else is
		// discussion, replies to applicants, or off-topic noise.
		if hit.ParentID != storyID {
			continue
		}
		raw := string(hit.CommentText)
		if !goKeyword.MatchString(raw) {
			continue
		}

		company, title, location := parseHNHeader(raw)
		if company == "" {
			continue
		}

		text := stripHTML(raw)
		if len(text) < 30 {
			continue
		}
		if location == "" {
			location = "Remote"
		}

		desc := text
		if len(desc) > 3000 {
			desc = desc[:3000] + "..."
		}

		var postedAt *time.Time
		if t, err := time.Parse(time.RFC3339, hit.CreatedAt); err == nil {
			postedAt = &t
		}

		job := &models.Job{
			Title:       title,
			Company:     company,
			SourceURL:   fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID),
			SourceBoard: "hnhiring",
			Description: desc,
			Location:    location,
			Country:     "Worldwide", // HN hiring threads are global; scheduler's US filter handles the rest.
			PostedAt:    postedAt,
		}
		NormalizeJob(job)
		jobs = append(jobs, *job)
	}

	log.Printf("[Scraper] HNHiring: fetched %d jobs", len(jobs))
	return jobs, nil
}

type hnHit struct {
	ObjectID    string `json:"objectID"`
	CommentText string `json:"comment_text"`
	ParentID    int64  `json:"parent_id"`
	CreatedAt   string `json:"created_at"`
	Author      string `json:"author"`
}

func (s *HNHiringScraper) get(rawURL string, out interface{}) error {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hnhiring: HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// latestHiringStory returns the item id of the most recent "Ask HN: Who is
// hiring?" thread, skipping the "Who wants to be hired?" sibling.
func (s *HNHiringScraper) latestHiringStory() (int64, error) {
	apiURL := "https://hn.algolia.com/api/v1/search_by_date" +
		"?tags=story,author_whoishiring&hitsPerPage=10"

	var result struct {
		Hits []struct {
			ObjectID string `json:"objectID"`
			Title    string `json:"title"`
		} `json:"hits"`
	}
	if err := s.get(apiURL, &result); err != nil {
		return 0, fmt.Errorf("hnhiring story lookup: %w", err)
	}

	for _, h := range result.Hits {
		if strings.HasPrefix(h.Title, "Ask HN: Who is hiring?") {
			if id, err := strconv.ParseInt(h.ObjectID, 10, 64); err == nil {
				return id, nil
			}
		}
	}
	return 0, fmt.Errorf("hnhiring: no hiring thread found in %d hits", len(result.Hits))
}

func (s *HNHiringScraper) threadComments(storyID int64) ([]hnHit, error) {
	apiURL := "https://hn.algolia.com/api/v1/search_by_date" +
		"?tags=" + url.QueryEscape("comment,story_"+strconv.FormatInt(storyID, 10)) +
		"&hitsPerPage=1000"

	var result struct {
		Hits []hnHit `json:"hits"`
	}
	if err := s.get(apiURL, &result); err != nil {
		return nil, fmt.Errorf("hnhiring comments: %w", err)
	}
	return result.Hits, nil
}

// parseHNHeader pulls company, role and location out of the conventional HN
// posting header, which is pipe-delimited: `Company | Role | Location | ...`.
//
// The header has to be read from the raw comment text: stripHTML collapses all
// whitespace with strings.Fields, so by the time a comment is stripped its line
// breaks are gone and the whole posting looks like one header line. Algolia
// separates paragraphs with <p>, so that becomes the line break here.
//
// Freeform postings that don't follow the convention are rejected rather than
// guessed at — the first line of a prose post is a sentence, not a company name.
func parseHNHeader(raw string) (company, title, location string) {
	firstLine := ""
	for _, line := range strings.Split(strings.ReplaceAll(raw, "<p>", "\n"), "\n") {
		line = stripHTML(line)
		if line != "" {
			firstLine = line
			break
		}
	}
	if firstLine == "" {
		return "", "", ""
	}

	// Some posters put the company on its own line above the header.
	parts := strings.Split(firstLine, "|")
	if len(parts) < 2 {
		return "", "", ""
	}

	company = cleanHNField(parts[0])
	title = cleanHNField(parts[1])
	if len(parts) > 2 {
		location = cleanHNField(parts[2])
	}

	if company == "" || title == "" || len(title) > 120 {
		return "", "", ""
	}
	// A colon in the company cell means the poster used freeform
	// `Field: value` lines ("Location: London, UK | Remote: Yes") instead of
	// the convention.
	if strings.Contains(company, ":") || !looksLikeRole(title) {
		return "", "", ""
	}
	return company, title, location
}

// roleWord matches the job-title vocabulary. Used as a positive filter: a
// second column that isn't a role is a posting written in a different column
// order (`NetBird | Berlin, Germany | Full-time`), and storing "Berlin,
// Germany" as the title is worse than dropping the row. Generic placeholders
// like "Multiple Positions" fail this on purpose — they say nothing about the
// role and the feed filters on Go/Golang anyway.
var roleWord = regexp.MustCompile(`(?i)\b(engineer|developer|programmer|architect|scientist|analyst|designer|manager|director|lead|founder|intern|sre|devops|sre|specialist|consultant|administrator|researcher|officer|staff|principal|cto|ceo|head of)\b`)

func looksLikeRole(s string) bool {
	return roleWord.MatchString(s)
}

// cleanHNField trims a header cell and strips a trailing URL or parenthetical
// so `Tether (https://tether.io/)` becomes `Tether`.
func cleanHNField(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ":")
	if open := strings.Index(s, "("); open > 0 {
		s = strings.TrimSpace(s[:open])
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, "-"))
	if len(s) > 80 {
		s = strings.TrimSpace(s[:80])
	}
	return s
}
