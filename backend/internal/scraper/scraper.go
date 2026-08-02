package scraper

import (
	"crypto/sha256"
	"fmt"
	"remotehunter/internal/models"
	"time"
)

// Scraper is the interface all job board scrapers must implement
type Scraper interface {
	Name() string
	Scrape(targetURL string) ([]models.Job, error)
}

// ComputeJobHash creates a SHA256 deduplication hash for a job
func ComputeJobHash(title, company, sourceURL string) string {
	h := sha256.New()
	h.Write([]byte(title + "|" + company + "|" + sourceURL))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NormalizeJob sets defaults and computes the hash for a scraped job
func NormalizeJob(job *models.Job) {
	job.ScrapedAt = time.Now()
	job.IsActive = true
	job.JobHash = ComputeJobHash(job.Title, job.Company, job.SourceURL)
}
