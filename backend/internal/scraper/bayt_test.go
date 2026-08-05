package scraper

import (
	"fmt"
	"testing"
)

func TestBaytScraper(t *testing.T) {
	s := NewBaytScraper()
	jobs, err := s.Scrape("https://www.bayt.com/en/international/jobs/golang-remote-jobs/")
	if err != nil {
		t.Fatalf("Bayt scrape failed: %v", err)
	}

	fmt.Printf("Bayt: Scraped %d jobs\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr)
	}

	if len(jobs) == 0 {
		t.Log("Warning: No bayt jobs fetched.")
	}
}
