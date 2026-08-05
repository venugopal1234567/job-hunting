package scraper

import (
	"fmt"
	"testing"
)

func TestZipRecruiterScraper(t *testing.T) {
	s := NewZipRecruiterScraper()
	jobs, err := s.Scrape("https://www.ziprecruiter.com/Jobs/Golang?location=Remote")
	if err != nil {
		t.Fatalf("ZipRecruiter scrape failed: %v", err)
	}

	fmt.Printf("ZipRecruiter: Scraped %d jobs\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr)
	}

	if len(jobs) == 0 {
		t.Log("Warning: No ziprecruiter jobs fetched.")
	}
}
