# RemoteHunter Implementation Tasks

## Phase 1: Infrastructure
- [x] docker-compose.yml

## Phase 2: Backend
- [x] go.mod
- [x] Dockerfile
- [x] internal/config/config.go
- [x] internal/db/database.go
- [x] internal/db/migrate.go
- [x] internal/models/models.go
- [x] internal/api/router.go
- [x] internal/api/handlers.go
- [x] internal/scraper/scraper.go (interface)
- [x] internal/scraper/golangprojects.go
- [x] internal/scraper/hnhiring.go
- [x] internal/scraper/weworkremotely.go
- [x] internal/scraper/scheduler.go
- [x] internal/resume/parser.go
- [x] internal/resume/skills.go
- [x] internal/ai/ollama.go
- [x] cmd/server/main.go

## Phase 3: Frontend (ui/)
- [x] package.json / vite.config / tailwind.config
- [x] src/types/index.ts
- [x] src/services/api.ts
- [x] src/hooks/useJobs.ts
- [x] src/hooks/useResume.ts
- [x] src/hooks/useAtsAnalysis.ts
- [x] src/components/layout/Navbar.tsx
- [x] src/components/dashboard/StatsOverview.tsx
- [x] src/components/dashboard/JobFilterBar.tsx
- [x] src/components/dashboard/JobCard.tsx
- [x] src/components/job-detail/JobDetailModal.tsx
- [x] src/components/resume/ResumeUploader.tsx
- [x] src/components/settings/SourceManager.tsx
- [x] src/App.tsx
- [x] Dockerfile (ui)
