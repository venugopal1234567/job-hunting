package scraper

import (
	"fmt"
	"testing"
)

func TestIndeedScraper(t *testing.T) {
	s := NewIndeedScraper()
	jobs, err := s.Scrape("https://apis.indeed.com/graphql?what=golang")
	if err != nil {
		t.Fatalf("Indeed scrape failed: %v", err)
	}

	fmt.Printf("Indeed: Scraped %d jobs\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr)
	}

	if len(jobs) == 0 {
		t.Log("Warning: No indeed jobs fetched.")
	}
}
