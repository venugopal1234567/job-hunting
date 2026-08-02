package scraper

import (
	"fmt"
	"testing"
	"time"
)

func TestBuiltInWithDaysSinceUpdated(t *testing.T) {
	s := NewBuiltInScraper()
	jobs, err := s.Scrape("https://builtin.com/jobs/remote/senior?search=Go&country=IND&allLocations=true&daysSinceUpdated=3")
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	fmt.Printf("Scraped %d jobs with daysSinceUpdated=3\n", len(jobs))
	now := time.Now()
	for i, j := range jobs {
		postedStr := "NIL"
		diffHours := -1.0
		if j.PostedAt != nil {
			postedStr = j.PostedAt.Format("2006-01-02 15:04:05")
			diffHours = now.Sub(*j.PostedAt).Hours()
		}
		fmt.Printf("[%d] Title: %s | Company: %s | PostedAt: %s (%.1f hours / %.1f days ago)\n",
			i+1, j.Title, j.Company, postedStr, diffHours, diffHours/24.0)
	}
}
