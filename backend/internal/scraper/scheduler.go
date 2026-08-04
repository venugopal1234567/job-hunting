package scraper

import (
	"database/sql"
	"log"
	"remotehunter/internal/models"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages background cron-based scraping
type Scheduler struct {
	cron      *cron.Cron
	db        *sql.DB
	scrapers  map[string]Scraper
	isRunning bool
}

// NewScheduler creates a new scheduler with all registered scrapers.
// Keys must match the board_name values stored in scraper_configs (case-insensitive).
func NewScheduler(db *sql.DB) *Scheduler {
	scrapers := map[string]Scraper{
		"golangprojects":   NewGolangProjectsScraper(),
		"hnhiring":         NewHNHiringScraper(),
		"weworkremotely":   NewWeWorkRemotelyScraper(),
		"remotive":         NewRemotiveScraper(),
		"arbeitnow":        NewArbeitnowScraper(),
		"remoteok":         NewRemoteOKScraper(),
		"builtin":          NewBuiltInScraper(),
		"builtinremote":    NewBuiltInScraper(),
		"flexboard":        NewFlexboardScraper(),
		"vacancyglobalpro": NewVacancyGlobalProScraper(),
	}

	return &Scheduler{
		cron:     cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger))),
		db:       db,
		scrapers: scrapers,
	}
}

// Start begins the cron scheduling based on database configs
func (s *Scheduler) Start() {
	configs, err := s.loadConfigs()
	if err != nil {
		log.Printf("[Scheduler] Failed to load configs: %v", err)
		return
	}

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		boardName := cfg.BoardName
		targetURL := cfg.TargetURL
		schedule := cfg.CronSchedule
		if schedule == "" {
			schedule = "@every 1h"
		}

		s.cron.AddFunc(schedule, func() {
			s.RunScraper(boardName, targetURL)
		})

		log.Printf("[Scheduler] Registered scraper '%s' → key='%s' schedule='%s'", boardName, normalizeKey(boardName), schedule)
	}

	s.cron.Start()
	s.isRunning = true
	log.Println("[Scheduler] Background cron scheduler started")

	// Run all scrapers immediately on startup
	go s.TriggerAll()
}

// Stop stops the background scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.isRunning = false
	log.Println("[Scheduler] Scheduler stopped")
}

// TriggerAll runs all enabled scrapers immediately
func (s *Scheduler) TriggerAll() {
	s.PurgeStaleJobs()
	configs, err := s.loadConfigs()
	if err != nil {
		log.Printf("[Scheduler] TriggerAll: failed to load configs: %v", err)
		return
	}

	for _, cfg := range configs {
		if cfg.Enabled {
			go s.RunScraper(cfg.BoardName, cfg.TargetURL)
		}
	}
}

// PurgeStaleJobs deletes jobs older than 30 days from the database
func (s *Scheduler) PurgeStaleJobs() {
	cutoff := time.Now().AddDate(0, 0, -30)
	res, err := s.db.Exec(`
		DELETE FROM jobs 
		WHERE (posted_at IS NOT NULL AND posted_at < $1)
		   OR (posted_at IS NULL AND scraped_at < $1)
	`, cutoff)
	if err != nil {
		log.Printf("[Scheduler] Error purging stale jobs: %v", err)
	} else if rows, _ := res.RowsAffected(); rows > 0 {
		log.Printf("[Scheduler] Purged %d stale jobs (>30 days old)", rows)
	}
}

// RunScraper executes a named scraper and upserts results to the database
func (s *Scheduler) RunScraper(boardName, targetURL string) {
	s.PurgeStaleJobs()

	key := normalizeKey(boardName)
	sc, ok := s.scrapers[key]
	if !ok {
		log.Printf("[Scheduler] No scraper registered for '%s' (key: '%s') — skipping", boardName, key)
		s.updateLastRun(boardName)
		return
	}

	log.Printf("[Scheduler] Running scraper '%s'", boardName)
	jobs, err := sc.Scrape(targetURL)
	if err != nil {
		log.Printf("[Scheduler] Scraper '%s' error: %v", boardName, err)
		return
	}

	saved := 0
	for _, job := range jobs {
		// Filter out US / USA jobs as requested by the user
		c := strings.ToLower(job.Country)
		l := strings.ToLower(job.Location)
		if c == "us" || c == "usa" || strings.Contains(c, "united states") || strings.Contains(c, "us only") ||
			l == "us" || l == "usa" || strings.Contains(l, "united states") || strings.Contains(l, "us only") {
			continue
		}

		if err := s.upsertJob(&job); err != nil {
			log.Printf("[Scheduler] Failed to save job '%s': %v", job.Title, err)
		} else {
			saved++
		}
	}

	log.Printf("[Scheduler] '%s': saved %d/%d jobs", boardName, saved, len(jobs))
	s.updateLastRun(boardName)
}

// upsertJob saves a job to the database, skipping if hash already exists
func (s *Scheduler) upsertJob(job *models.Job) error {
	cleanTitle := truncateStr(strings.ToValidUTF8(strings.ReplaceAll(job.Title, "\x00", ""), ""), 250)
	cleanCompany := truncateStr(strings.ToValidUTF8(strings.ReplaceAll(job.Company, "\x00", ""), ""), 250)
	cleanDesc := strings.ToValidUTF8(strings.ReplaceAll(job.Description, "\x00", ""), "")
	cleanLocation := truncateStr(job.Location, 250)
	cleanCountry := truncateStr(job.Country, 90)
	cleanBoard := truncateStr(job.SourceBoard, 90)

	_, err := s.db.Exec(`
		INSERT INTO jobs (job_hash, title, company, location, country, source_url, source_board, description, salary_range, job_type, posted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (job_hash) DO UPDATE SET
			description = CASE WHEN LENGTH(EXCLUDED.description) > LENGTH(jobs.description) THEN EXCLUDED.description ELSE jobs.description END,
			source_url = EXCLUDED.source_url,
			posted_at = EXCLUDED.posted_at`,
		job.JobHash, cleanTitle, cleanCompany, cleanLocation, cleanCountry,
		job.SourceURL, cleanBoard, cleanDesc, job.SalaryRange,
		job.JobType, job.PostedAt,
	)
	return err
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// loadConfigs reads scraper configurations from database
func (s *Scheduler) loadConfigs() ([]models.ScraperConfig, error) {
	rows, err := s.db.Query(`SELECT id, board_name, target_url, enabled, cron_schedule, last_run_at FROM scraper_configs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.ScraperConfig
	for rows.Next() {
		var c models.ScraperConfig
		var lastRunAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.BoardName, &c.TargetURL, &c.Enabled, &c.CronSchedule, &lastRunAt); err != nil {
			continue
		}
		if lastRunAt.Valid {
			c.LastRunAt = &lastRunAt.Time
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (s *Scheduler) updateLastRun(boardName string) {
	s.db.Exec(`UPDATE scraper_configs SET last_run_at = $1 WHERE board_name = $2`, time.Now(), boardName)
}

// normalizeKey converts a board_name from the DB to the scrapers map key.
// All keys in the scrapers map are lowercase, no spaces.
func normalizeKey(name string) string {
	// Strip spaces, lowercase
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}
