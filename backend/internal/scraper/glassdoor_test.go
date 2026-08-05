package scraper

import (
	"fmt"
	"testing"
)

func TestGlassdoorScraper(t *testing.T) {
	s := NewGlassdoorScraper()
	jobs, err := s.Scrape("https://www.glassdoor.co.in/Job/india-golang-jobs-SRCH_IL.0,5_KO6,12.htm?remoteWorkType=1")
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	fmt.Printf("Scraped %d jobs from Glassdoor\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s | Desc length: %d\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr, len(j.Description))
	}

	if len(jobs) == 0 {
		t.Log("Warning: No Glassdoor jobs found.")
	}
}
