package scraper

import (
	"fmt"
	"testing"
)

func TestWeWorkRemotelyHTML(t *testing.T) {
	s := NewWeWorkRemotelyScraper()
	jobs, err := s.Scrape("https://weworkremotely.com/remote-jobs-golang")
	if err != nil {
		t.Fatalf("HTML Scrape error: %v", err)
	}

	fmt.Printf("Scraped %d jobs from WeWorkRemotely HTML page\n", len(jobs))
	for i, j := range jobs {
		postedStr := "NIL"
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02")
		}
		fmt.Printf("[%d] Title: %s | Company: %s | Location: %s | Country: %s | Posted: %s | Desc length: %d\n",
			i+1, j.Title, j.Company, j.Location, j.Country, postedStr, len(j.Description))
	}
}
