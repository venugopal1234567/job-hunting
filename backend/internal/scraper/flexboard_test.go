package scraper

import (
	"fmt"
	"testing"
)

func TestFlexboardScraper(t *testing.T) {
	s := NewFlexboardScraper()
	jobs, err := s.Scrape("https://flexboard.9y.liveblog365.com/?search=golang")
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	if len(jobs) == 0 {
		t.Fatalf("No jobs scraped")
	}

	fmt.Printf("Scraped %d jobs from FlexBoard:\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | SourceURL: %s | PostedAt: %s\n",
			i+1, j.Title, j.Company, j.Location, j.Country, j.SourceURL, postedStr)
		if j.Description == "" || len(j.Description) < 50 {
			t.Errorf("Job index %d has empty or too short description: %q", i, j.Description)
		}
	}
}
