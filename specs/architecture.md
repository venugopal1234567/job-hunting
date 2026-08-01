# RemoteHunter System Architecture Specification

## 1. System Overview
RemoteHunter is a full-stack automated job search, matching, and AI-powered ATS resume preparation system running as a continuous background containerized service.

```
+-----------------------------------------------------------------------------------+
|                                  RemoteHunter App                                 |
|                                                                                   |
|  +-------------------+       +--------------------+       +--------------------+  |
|  |  React Frontend   | <---> |   Go Backend API   | <---> | PostgreSQL DB      |  |
|  |  (Vite+TS+Tailwind|       |  (Gin Router &     |       | (Port 5433:5432)   |  |
|  |   Dashboard & UI) |       |   Cron Scheduler)  |       +--------------------+  |
|  +-------------------+       +---------+----------+                               |
|                                        |                                          |
|                                        +----------> Modular Scrapers              |
|                                        |            - GolangProjects              |
|                                        |            - HN Hiring (Go)              |
|                                        |            - BuiltIn Remote              |
|                                        |            - WeWorkRemotely              |
|                                        |            - WelcomeToTheJungle          |
|                                        |                                          |
|                                        +----------> Go Resume Parser (PDF/Text)   |
|                                        |                                          |
|                                        +----------> Local Ollama / Gemma 4 AI API |
+-----------------------------------------------------------------------------------+
```

## 2. Component Design

### 2.1 Backend (Go + Gin)
- **Gin Router**: Serves RESTful JSON APIs for job search, settings, resume processing, and ATS analysis.
- **Scraper Engine (`pkg/scraper`)**: Interface-driven web scraping framework. Dedicated modular scrapers & parsers for target URLs:
  1. `GolangProjects`: https://www.golangprojects.com/golang-remote-jobs.html
  2. `HNHiring`: https://hnhiring.com/technologies/go
  3. `BuiltIn`: https://builtin.com/jobs/remote/senior (dynamic filters: skills, daysSinceUpdated, country)
  4. `WeWorkRemotely`: https://weworkremotely.com/remote-jobs/search
  5. `WelcomeToTheJungle`: https://app.welcometothejungle.com/
- **Scheduler (`pkg/scheduler`)**: Go `robfig/cron/v3` or native ticker for periodic background scraping with SHA-256 deduplication.
- **Resume Processor (`pkg/resume`)**: PDF/text extraction using `ledongthuc/pdf` and string processing for skill extraction against skill taxonomy.
- **Ollama AI Client (`pkg/ai`)**: REST HTTP client targeting local Ollama service (`http://host.docker.internal:11434` or custom Ollama URL) requesting `gemma4` / `gemma2` formatted structured JSON responses.

### 2.2 Database (PostgreSQL)
- Runs inside Docker container, host port `5433` mapped to container port `5432`.
- Persistent data volume for job records, candidate resumes, settings, and generated AI ATS reports.

### 2.3 Frontend (React + Vite + TypeScript + Tailwind CSS)
- **Dashboard**: Main view with job feed, match scores, filtering (skills, date, country).
- **ATS Analysis Modal / Detail View**: Displays ATS match score breakdown, actionable resume bullet points, and 2-3 interactive clarifying skill-gap questions.
- **Settings Panel**: Configure job board target URLs, dynamic search keywords, scraping interval, and location filters.
- **Resume Manager**: Upload, preview, and extract skills from master resume.

### 2.4 Docker & Continuous Execution
- Managed via `docker-compose.yml` with `restart: unless-stopped`.
- Services: `db` (Postgres), `backend` (Go API), `frontend` (React Nginx container).
