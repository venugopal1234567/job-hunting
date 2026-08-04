# Guide to Adding a New Scraper

This guide details the step-by-step process for adding a new job board scraper to the RemoteHunter application, covering changes required across both the Backend and Frontend.

---

## 1. Create the Scraper File
Create a new scraper implementation file under `backend/internal/scraper/<board_name>.go`.

Implement the `Scraper` interface:
```go
type Scraper interface {
	Name() string
	Scrape(targetURL string) ([]models.Job, error)
}
```

### Example Boilerplate:
```go
package scraper

import (
	"fmt"
	"net/http"
	"remotehunter/internal/models"
	"time"
)

type MyNewScraper struct {
	client *http.Client
}

func NewMyNewScraper() *MyNewScraper {
	return &MyNewScraper{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *MyNewScraper) Name() string { return "mynewscraper" }

func (s *MyNewScraper) Scrape(targetURL string) ([]models.Job, error) {
	if targetURL == "" {
		targetURL = "https://example.com/jobs"
	}
	
	// 1. Fetch HTML/JSON page content
	// 2. Parse listing cards (concurrently fetch detail pages if required)
	// 3. Populate models.Job objects:
	//    job := models.Job{
	//        Title:       "Title",
	//        Company:     "Company",
	//        Location:    "Remote",
	//        Country:     inferCountry("Remote"), // standard normalizer
	//        SourceURL:   detailURL,
	//        SourceBoard: "mynewscraper",
	//        Description: "...",
	//    }
	// 4. Run NormalizeJob(&job) to assign hashes and compute values
	
	return jobs, nil
}
```

---

## 2. Register Scraper in Scheduler
Add your new scraper to the registry map in [scheduler.go](file:///home/venu/Documents/projects/ai/job-hunting/backend/internal/scraper/scheduler.go):

```diff
 func NewScheduler(db *sql.DB) *Scheduler {
 	scrapers := map[string]Scraper{
 		"golangprojects":   NewGolangProjectsScraper(),
 		"flexboard":        NewFlexboardScraper(),
 		"vacancyglobalpro": NewVacancyGlobalProScraper(),
+		"mynewscraper":     NewMyNewScraper(),
 	}
```

*Note: Ensure the map key is strictly lowercase and alphanumeric (matching the DB `board_name` stripped of spaces and lowercased).*

---

## 3. Seed Scraper Config in DB Migrations
Register the default configuration and scrape schedule in [migrate.go](file:///home/venu/Documents/projects/ai/job-hunting/backend/internal/db/migrate.go):

```diff
     ('FlexBoard', 'https://flexboard.9y.liveblog365.com/?search=golang', true, '@every 1h'),
-    ('VacancyGlobalPro', 'https://vacancyglobalpro.up.railway.app/remote-golang-jobs', true, '@every 1h')
+    ('VacancyGlobalPro', 'https://vacancyglobalpro.up.railway.app/remote-golang-jobs', true, '@every 1h'),
+    ('MyNewScraper', 'https://example.com/jobs', true, '@every 1h')
 ON CONFLICT (board_name) DO NOTHING;
```

To update an active running database instance immediately without resetting migrations:
```sql
INSERT INTO scraper_configs (board_name, target_url, enabled, cron_schedule)
VALUES ('MyNewScraper', 'https://example.com/jobs', true, '@every 1h')
ON CONFLICT (board_name) DO NOTHING;
```

---

## 4. Add Filter Pill in the UI
To let users filter specifically by your new source:

1. Add your new source to the `Scraped Sources` pill list in [JobFilterBar.tsx](file:///home/venu/Documents/projects/ai/job-hunting/ui/src/components/dashboard/JobFilterBar.tsx):

```diff
           <div className="flex flex-wrap gap-1 mt-1 mb-2">
             {[
               { label: 'FlexBoard', value: 'flexboard' },
               { label: 'BuiltIn', value: 'builtin' },
               { label: 'VacancyPro', value: 'vacancyglobalpro' },
+              { label: 'MyNewScraper', value: 'mynewscraper' },
               ...
```

---

## 5. Rebuild and Deploy
Apply changes by rebuilding the Docker stack:
```bash
docker compose up -d --build backend ui
```
