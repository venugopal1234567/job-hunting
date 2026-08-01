# RemoteHunter Database Schema Specification

## PostgreSQL Database Configuration
- **Container Name**: `remotehunter-db`
- **Host Port**: `5433`
- **Container Port**: `5432`
- **Database Name**: `remotehunter`
- **User / Password**: `hunter / hunterpass`

## Schema Tables

```sql
-- Create extension for UUID generation if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Jobs Table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256 of title + company + source_url for deduplication
    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    country VARCHAR(100),
    source_url TEXT NOT NULL,
    source_board VARCHAR(100) NOT NULL, -- e.g., 'golangprojects', 'hnhiring', 'builtin', 'weworkremotely', 'welcometothejungle'
    description TEXT NOT NULL,
    salary_range VARCHAR(100),
    job_type VARCHAR(50), -- e.g., 'Full-time', 'Contract'
    posted_at TIMESTAMP WITH TIME ZONE,
    scraped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Indexing for rapid query filtering
CREATE INDEX idx_jobs_posted_at ON jobs(posted_at DESC);
CREATE INDEX idx_jobs_country ON jobs(country);
CREATE INDEX idx_jobs_source_board ON jobs(source_board);

-- 2. Skills Taxonomy Table
CREATE TABLE IF NOT EXISTS skills (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    category VARCHAR(100) -- e.g., 'Language', 'Framework', 'Database', 'DevOps', 'Tool'
);

-- 3. Job Required Skills Table (Many-to-Many)
CREATE TABLE IF NOT EXISTS job_skills (
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    skill_id INT REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (job_id, skill_id)
);

-- 4. Candidate Resumes Table
CREATE TABLE IF NOT EXISTS resumes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    filename VARCHAR(255) NOT NULL,
    raw_text TEXT NOT NULL,
    extracted_skills JSONB, -- Array of strings e.g. ["Go", "PostgreSQL", "Docker"]
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- 5. ATS Analysis Reports Table
CREATE TABLE IF NOT EXISTS ats_analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    resume_id UUID REFERENCES resumes(id) ON DELETE CASCADE,
    ats_score INT NOT NULL CHECK (ats_score >= 0 AND ats_score <= 100),
    match_breakdown JSONB, -- { "matched_skills": [], "missing_skills": [] }
    actionable_suggestions JSONB, -- ["bullet point 1", "bullet point 2"]
    gap_questions JSONB, -- [{"question": "...", "skill": "CI/CD"}]
    analyzed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(job_id, resume_id)
);

-- 6. Scraper Configurations Table
CREATE TABLE IF NOT EXISTS scraper_configs (
    id SERIAL PRIMARY KEY,
    board_name VARCHAR(100) UNIQUE NOT NULL,
    target_url TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    cron_schedule VARCHAR(50) DEFAULT '@every 1h',
    last_run_at TIMESTAMP WITH TIME ZONE
);

-- Default seeds for required target remote job boards
INSERT INTO scraper_configs (board_name, target_url, enabled, cron_schedule)
VALUES 
    ('GolangProjects', 'https://www.golangprojects.com/golang-remote-jobs.html', true, '@every 1h'),
    ('HNHiring', 'https://hnhiring.com/technologies/go', true, '@every 1h'),
    ('BuiltIn Remote', 'https://builtin.com/jobs/remote/senior?search=Go&daysSinceUpdated=7&skills=Go%2CPython%2CAWS%2CDocker%2CGCP%2CTypescript%2CAzure%2CCi%2FCd%2CPostgres%2CRust%2CSQL%2CNode.js&country=IND&allLocations=true', true, '@every 1h'),
    ('WeWorkRemotely', 'https://weworkremotely.com/remote-jobs/search?sort=Past+24+Hours', true, '@every 1h'),
    ('WelcomeToTheJungle', 'https://app.welcometothejungle.com/', true, '@every 1h')
ON CONFLICT (board_name) DO UPDATE SET target_url = EXCLUDED.target_url;
```
