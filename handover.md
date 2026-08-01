# RemoteHunter — Handover Document

## 1. Project Overview & Architecture
**RemoteHunter** is an AI-powered job search and ATS resume matching engine for remote tech roles.

* **Backend:** Go (Gin framework), PostgreSQL 16 database, `robfig/cron/v3` background scheduler, `chromedp` (Headless Chromium) for browser-rendered scraping, and local Ollama (`gemma4:e4b`) integration.
* **Frontend:** React 18, TypeScript, Tailwind CSS, Lucide icons, Vite build system.
* **Infrastructure:** Docker Compose orchestration (`db`, `backend`, `ui`).

---

## 2. Directory Structure & Key Files

```
job-hunting/
├── docker-compose.yml              # Container orchestration (DB port 5433, Backend 8080, UI 3000)
├── specs/                          # Specification documents (api_spec, architecture, etc.)
├── backend/
│   ├── Dockerfile                  # Alpine + Go 1.22 + Chromium & Chromedriver
│   ├── go.mod / go.sum             # Go module dependencies
│   ├── cmd/server/main.go          # Application entrypoint
│   └── internal/
│       ├── config/config.go        # Env configuration (DB, Ollama host & model)
│       ├── db/
│       │   ├── database.go         # Postgres connection pool
│       │   └── migrate.go          # DDL migrations & default scraper/skill seeds
│       ├── models/models.go        # Shared domain structs (Job, Resume, ATSAnalysis, etc.)
│       ├── ai/ollama.go            # Ollama client for Gemma 4 AI matching & ATS evaluation
│       ├── resume/
│       │   ├── parser.go           # PDF & raw text resume parser
│       │   └── skills.go           # 50+ skill taxonomy extractor
│       ├── scraper/
│       │   ├── scraper.go          # Scraper interface & job hash deduplication
│       │   ├── remotive.go         # Remotive JSON API scraper
│       │   ├── arbeitnow.go        # Arbeitnow JSON API scraper
│       │   ├── weworkremotely.go   # WeWorkRemotely RSS feed parser
│       │   ├── golangprojects.go   # GolangProjects HTML scraper
│       │   ├── hnhiring.go         # HNHiring HTML scraper
│       │   ├── builtin.go          # BuiltIn chromedp (headless browser) scraper
│       │   └── scheduler.go        # Scraper runner & database upsert with UTF-8 cleaning
│       └── api/
│           ├── router.go           # Gin router configuration & CORS
│           └── handlers.go         # REST API route handlers
└── ui/
    ├── Dockerfile                  # Multi-stage Node.js build -> Nginx server
    ├── package.json / vite.config.ts
    ├── index.html
    └── src/
        ├── App.tsx                 # Root layout & tab router
        ├── types/index.ts          # TypeScript models matching backend schema
        ├── services/api.ts         # Axios service layer for REST endpoints
        ├── hooks/                  # Custom hooks (useJobs, useResume, useAtsAnalysis)
        └── components/
            ├── layout/Navbar.tsx
            ├── dashboard/
            │   ├── StatsOverview.tsx
            │   ├── JobFilterBar.tsx
            │   └── JobCard.tsx
            ├── job-detail/JobDetailModal.tsx
            ├── resume/ResumeUploader.tsx
            └── settings/SourceManager.tsx
```

---

## 3. Database Schema (`remotehunter`)

* `jobs`: Stores deduplicated job listings (`id`, `job_hash`, `title`, `company`, `location`, `country`, `source_url`, `source_board`, `description`, `salary_range`, `job_type`, `posted_at`, `scraped_at`, `is_active`).
* `skills`: Skill taxonomy lookup table (`id`, `name`, `category`).
* `job_skills`: Many-to-many relationship mapping jobs to extracted skills.
* `resumes`: Stores uploaded user resumes (`id`, `user_id`, `filename`, `raw_text`, `extracted_skills`, `created_at`).
* `ats_analyses`: Caches Ollama/AI ATS match reports per job and resume (`id`, `job_id`, `resume_id`, `ats_score`, `match_breakdown`, `suggestions`, `gap_questions`, `created_at`).
* `scraper_configs`: Manages scraping targets and cron frequencies (`id`, `board_name`, `target_url`, `enabled`, `cron_schedule`, `last_run_at`).

---

## 4. Implemented REST API Endpoints (`/api/v1`)

* `GET /health` — Backend health check.
* `GET /jobs` — Query filtered jobs (supports parameters: `skills`, `days`, `country`, `page`, `limit`).
* `GET /jobs/:id` — Get single job details with cached ATS analysis.
* `POST /jobs/trigger-scrape` — Trigger manual scrape across all enabled scraper sources.
* `POST /jobs/:id/analyze` — Trigger Ollama Gemma 4 ATS match analysis for a job against the active resume.
* `POST /resume/upload` — Multipart form upload for `.pdf` or `.txt` resumes.
* `GET /resume/active` — Fetch current active user resume and extracted skills.
* `GET /settings` — Get scraper configurations.
* `PUT /settings/sources` — Update scraper target URLs and enable/disable states.

---

## 5. Current Scraper Status & Notes for Next Agent

1. **Working Scrapers:**
   - **Remotive API** (`remotive.go`): Fetches public JSON jobs.
   - **WeWorkRemotely RSS** (`weworkremotely.go`): Parses programming category RSS feed.
   - **Arbeitnow API** (`arbeitnow.go`): JSON API for remote roles.

2. **Scraper Under Iteration / Known Feedback:**
   - **BuiltIn Remote (`builtin.go`):** Implemented using `chromedp` (headless Chromium inside the backend Alpine container). The user indicated that the initial scraped results did not match the expected real-time listings on BuiltIn (e.g. recent Go roles in India like Kaseya / Arrow Electronics).
   - **Key Selectors Updated:** BuiltIn JS extractor in `builtin.go` was updated to target `[data-id="job-card"]` and `[data-id="job-card-title"]`.
   - **Task for Next Session:** Verify if `chromedp` requires additional wait times, user-agent adjustments, or specific page pagination parameters to ensure BuiltIn search filters (e.g. `country=IND`, `daysSinceUpdated=1`) properly render before DOM extraction.

---

## 6. How to Run & Verify

```bash
# Start all containers via Docker Compose
docker compose up --build -d

# Check running container status
docker compose ps

# Inspect backend logs
docker compose logs -f backend

# Frontend application access:
# http://localhost:3000

# Backend API access:
# http://localhost:8080/api/v1/health
```
