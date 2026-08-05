package scraper

import (
	"fmt"
	"testing"
)

func TestRealWorkFromAnywhereScraper(t *testing.T) {
	s := NewRealWorkFromAnywhereScraper()
	jobs, err := s.Scrape("https://www.realworkfromanywhere.com/remote-backend-jobs")
	if err != nil {
		t.Skipf("RealWorkFromAnywhere scrape skipped/failed: %v", err)
		return
	}

	fmt.Printf("Scraped %d Go/Golang jobs from RealWorkFromAnywhere\n", len(jobs))
	for i, j := range jobs {
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Desc length: %d\n",
			i+1, j.Title, j.Company, j.Location, j.Country, len(j.Description))
	}
}
