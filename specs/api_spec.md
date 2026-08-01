# RemoteHunter REST API Specification

Base URL: `http://localhost:8080/api/v1`

## 1. Job Endpoints

### `GET /jobs`
Fetch aggregated jobs with dynamic search filtering and match scores against current active resume.
- **Query Parameters**:
  - `skills`: comma-separated skills (e.g. `Go,Postgres,Docker`)
  - `days`: max days since updated (default `1`, customizable to `7`, `30`)
  - `country`: filter string (e.g., `Worldwide`, `US`, `Germany`)
  - `page`: integer (default 1)
  - `limit`: integer (default 20)
- **Response `200 OK`**:
```json
{
  "total": 42,
  "page": 1,
  "limit": 20,
  "jobs": [
    {
      "id": "c7a84e20-...",
      "title": "Senior Go Backend Engineer",
      "company": "TechCorp Inc.",
      "location": "Remote (India)",
      "country": "US",
      "source_url": "https://...",
      "source_board": "remotive",
      "description": "We are seeking a Go Engineer...",
      "salary_range": "$120,000 - $150,000",
      "posted_at": "2026-08-01T10:00:00Z",
      "matched_skills": ["Go", "Postgres"],
      "missing_skills": ["Kubernetes"],
      "ats_score": 85
    }
  ]
}
```

### `GET /jobs/:id`
Get detailed job posting along with cached ATS analysis report (if generated).

### `POST /jobs/trigger-scrape`
Trigger manual background scrape execution across all configured sources.

---

## 2. Resume & Skill Extraction Endpoints

### `POST /resume/upload`
Upload resume file (`multipart/form-data` with `file` field - `.pdf` or `.txt`).
- **Response `200 OK`**:
```json
{
  "id": "e9b72a11-...",
  "filename": "john_doe_resume.pdf",
  "extracted_skills": ["Go", "PostgreSQL", "Docker", "REST API", "Git", "Linux"],
  "raw_text_length": 3450,
  "uploaded_at": "2026-08-01T19:00:00Z"
}
```

### `GET /resume/active`
Retrieve currently active resume and extracted skill list.

---

## 3. Local AI (Gemma 4 / Ollama) ATS Analysis Endpoint

### `POST /jobs/:id/analyze`
Trigger local Gemma 4 AI analysis for a specific job against active resume.
- **Request Body**: (optional parameters, e.g., `{ "resume_id": "uuid" }`)
- **Response `200 OK`**:
```json
{
  "id": "f5120a22-...",
  "job_id": "c7a84e20-...",
  "ats_score": 88,
  "match_breakdown": {
    "matched_skills": ["Go", "PostgreSQL", "REST APIs", "Docker"],
    "missing_skills": ["gRPC", "CI/CD"]
  },
  "actionable_suggestions": [
    "Highlight your experience with high-throughput Go backend services in your summary.",
    "Add explicit metrics on database query optimizations using PostgreSQL."
  ],
  "gap_questions": [
    {
      "skill": "CI/CD",
      "question": "The job specifies automated CI/CD pipelines. Have you configured GitHub Actions or GitLab CI in any of your Go projects?"
    },
    {
      "skill": "gRPC",
      "question": "Did you work with protocol buffers or microservices communication even if not explicitly labeled as gRPC?"
    }
  ],
  "analyzed_at": "2026-08-01T19:15:00Z"
}
```

---

## 4. Settings & Configuration Endpoints

### `GET /settings`
Retrieve scraper sources, default skill keywords, and polling schedule.

### `PUT /settings/sources`
Add or update target job scrapers/URLs.
