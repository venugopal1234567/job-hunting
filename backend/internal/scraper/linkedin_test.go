package scraper

import (
	"fmt"
	"testing"
)

func TestLinkedInScraper(t *testing.T) {
	s := NewLinkedInScraper()
	
	// Test targeting remote Golang positions in India and Worldwide
	targetURL := "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=golang&location=India&f_WT=2|" +
		"https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=golang&location=Worldwide&f_WT=2"

	jobs, err := s.Scrape(targetURL)
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	fmt.Printf("Scraped %d total jobs from LinkedIn Guest API\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s | Desc length: %d\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr, len(j.Description))
	}

	if len(jobs) == 0 {
		t.Log("Warning: No jobs found. This might be due to transient blocking or search parameters, but should be verified.")
	}
}
