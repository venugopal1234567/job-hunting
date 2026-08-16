# RemoteHunter Backend — Developer & Agent Guidance

This document outlines the architectural conventions, file limits, package layout, and development guidelines for AI agents and human developers maintaining the Go backend (`backend/`).

---

## 1. Core Architecture Principles & Constraints

### A. Strict File Size Limit (< 500 Lines)
- **Rule**: No Go source file (`*.go`) in `internal/` or `cmd/` may exceed **500 lines**.
- **Target**: Keep files focused within **100–350 lines**.
- **Action**: When adding new handlers, services, or prompts, create a new modular file in the appropriate package rather than inflating existing files.

### B. Layered Architecture & Separation of Concerns

```
cmd/server/main.go
   ↓
internal/api (HTTP Handlers / Gin Controllers)
   ↓
internal/repository (Data Access Interfaces & PostgreSQL Implementations)
   ↓
internal/ai, internal/parser, internal/pdf, internal/scraper (Domain Engines)
```

1. **HTTP Handlers (`internal/api/`)**:
   - Responsibility: Parameter parsing, JSON decoding, status code handling, invoking repository/services, returning responses.
   - **Constraint**: Handlers MUST NOT perform raw SQL queries (`db.QueryRow`, `db.Exec`). All database access must pass through `internal/repository` interfaces.

2. **Repository Layer (`internal/repository/`)**:
   - Responsibility: Encapsulating database interaction behind Go interfaces.
   - Package `repository`: Declares abstract interfaces (`JobRepository`, `ResumeRepository`, `SettingsRepository`).
   - Package `repository/postgres`: Implements database queries matching `internal/db/migrate.go` table schemas.

3. **AI Provider & Prompt System (`internal/ai/`)**:
   - `client.go`: Core HTTP client, fallback model switching (NVIDIA API / OpenAI compatible endpoints).
   - `tailor.go`: ATS analysis and resume tailoring logic.
   - `converter.go`: Conversion of raw resumes to structured ATS JSON.
   - `template.go`: Single-page HTML resume rendering template engine.
   - `validator.go`: Senior recruiter audit and validation engine.
   - `prompts/`: **All prompt templates MUST reside in `internal/ai/prompts` as package constants**.

---

## 2. Directory Structure

```
backend/
├── cmd/
│   ├── server/main.go               # Main API entrypoint & dependency injection
│   └── fix_resume/main.go           # CLI utility script
├── internal/
│   ├── ai/                          # AI provider, HTML renderer, & parser
│   │   ├── client.go
│   │   ├── converter.go
│   │   ├── parser.go
│   │   ├── tailor.go
│   │   ├── template.go
│   │   ├── validator.go
│   │   └── prompts/                 # System prompt constants
│   │       ├── converter_prompts.go
│   │       ├── tailor_prompts.go
│   │       └── validator_prompts.go
│   ├── api/                         # HTTP Handlers (Gin)
│   │   ├── handler.go               # Base Handler struct with injected repos
│   │   ├── router.go                # Gin router setup & CORS configuration
│   │   ├── job_handler.go           # /api/v1/jobs endpoints
│   │   ├── resume_handler.go        # /api/v1/resume endpoints
│   │   ├── ai_handler.go            # /api/v1/ai & chat endpoints
│   │   └── settings_handler.go      # /api/v1/settings & health endpoints
│   ├── config/                      # Environment variable loading
│   ├── db/                          # Database connection & schema migrations
│   ├── models/                      # Domain models & JSON payloads
│   ├── parser/                      # Text parsing engine for resumes
│   ├── pdf/                         # Headless Chromium PDF generation service
│   ├── repository/                  # Data access interfaces & implementations
│   │   ├── repository.go            # Go interfaces
│   │   └── postgres/                # PostgreSQL implementation
│   │       ├── job_repo.go
│   │       ├── resume_repo.go
│   │       └── settings_repo.go
│   ├── resume/                      # Skill taxonomy & text cleanup utilities
│   └── scraper/                     # Job scrapers & background scheduler
├── Dockerfile
├── go.mod
└── go.sum
```

---

## 3. Database Schema Conventions (`internal/db/migrate.go`)

When adding or modifying queries in `internal/repository/postgres`:
- **`jobs` Table**: `id`, `job_hash`, `title`, `company`, `location`, `country`, `source_url`, `source_board`, `description`, `salary_range`, `job_type`, `posted_at`, `scraped_at`, `is_active`.
- **`resumes` Table**: `id`, `filename`, `raw_text`, `edited_text`, `extracted_skills`, `uploaded_at`, `is_active`.
- **`resume_versions` Table**: `id`, `resume_id`, `job_id`, `snapshot_text`, `label`, `applied_at`, `source`.

*Always verify exact column names against `internal/db/migrate.go` before altering repository queries.*

---

## 4. Verification Workflow for Changes

Before completing any task in `backend/`:
1. Check line counts for Go source files:
   ```bash
   find . -name "*.go" -exec wc -l {} + | sort -nr
   ```
   Ensure no file exceeds 500 lines.
2. Compile and run test suite:
   ```bash
   go build ./... && go test ./...
   ```
