package scraper

import (
	"fmt"
	"testing"
)

func TestRemoteRocketship(t *testing.T) {
	s := NewRemoteRocketshipScraper()
	jobs, err := s.Scrape("https://www.remoterocketship.com/?ref=yanirs-established-remote&page=1&sort=DateAdded&jobTitle=Golang&locations=Worldwide%2CIndia")
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	fmt.Printf("Scraped %d jobs from Remote Rocketship\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s | Desc length: %d\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr, len(j.Description))
	}
}
