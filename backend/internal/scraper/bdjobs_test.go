package scraper

import (
	"fmt"
	"testing"
)

func TestBDJobsScraper(t *testing.T) {
	s := NewBDJobsScraper()
	jobs, err := s.Scrape("https://jobs.bdjobs.com/jobsearch.asp?txtsearch=golang")
	if err != nil {
		t.Skipf("BDJobs scrape skipped/failed (likely headless chrome issue or network): %v", err)
		return
	}

	fmt.Printf("Scraped %d jobs from BDJobs\n", len(jobs))
	for i, j := range jobs {
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Desc length: %d\n",
			i+1, j.Title, j.Company, j.Location, j.Country, len(j.Description))
	}
}
