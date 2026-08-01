package db

import (
	"database/sql"
	"log"
)

// Migrate creates all required tables and seeds default data
func Migrate(db *sql.DB) error {
	schema := `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Jobs Table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_hash VARCHAR(64) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    country VARCHAR(100),
    source_url TEXT NOT NULL,
    source_board VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    salary_range VARCHAR(100),
    job_type VARCHAR(50),
    posted_at TIMESTAMP WITH TIME ZONE,
    scraped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_jobs_posted_at ON jobs(posted_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_country ON jobs(country);
CREATE INDEX IF NOT EXISTS idx_jobs_source_board ON jobs(source_board);

-- 2. Skills Taxonomy Table
CREATE TABLE IF NOT EXISTS skills (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    category VARCHAR(100)
);

-- 3. Job Required Skills (Many-to-Many)
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
    extracted_skills JSONB,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- 5. ATS Analysis Reports Table
CREATE TABLE IF NOT EXISTS ats_analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    resume_id UUID REFERENCES resumes(id) ON DELETE CASCADE,
    ats_score INT NOT NULL CHECK (ats_score >= 0 AND ats_score <= 100),
    match_breakdown JSONB,
    actionable_suggestions JSONB,
    gap_questions JSONB,
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

-- Seed default scraper targets
INSERT INTO scraper_configs (board_name, target_url, enabled, cron_schedule)
VALUES
    -- ✅ Free JSON APIs — work reliably from Docker (no JS rendering needed)
    ('Remotive', 'https://remotive.com/api/remote-jobs?search=golang&limit=50', true, '@every 1h'),
    ('Arbeitnow', 'https://www.arbeitnow.com/api/job-board-api?remote=true&page=1', true, '@every 1h'),
    -- ✅ RSS Feed — reliable XML endpoint
    ('WeWorkRemotely', 'https://weworkremotely.com/categories/remote-programming-jobs.rss', true, '@every 1h'),
    -- ✅ Headless Chromium (chromedp) — scrapes JS-rendered BuiltIn SPA
    ('BuiltIn Remote', 'https://builtin.com/jobs/remote/senior?search=Go&daysSinceUpdated=1&skills=Go%2CPython%2CAWS%2CDocker%2CGCP%2CTypescript%2CAzure%2CCi%2FCd%2CPostgres%2CRust%2CSQL%2CNode.js&country=IND&allLocations=true', true, '@every 2h'),
    -- ℹ️ HTML scrapers — may return 0 jobs if site blocks bots or uses JS rendering
    ('GolangProjects', 'https://www.golangprojects.com/golang-remote-jobs.html', true, '@every 2h'),
    ('HNHiring', 'https://hnhiring.com/technologies/go', true, '@every 2h'),
    -- ❌ JS-rendered SPA — no scraper implemented (requires headless browser)
    ('WelcomeToTheJungle', 'https://app.welcometothejungle.com/', false, '@every 6h')
ON CONFLICT (board_name) DO UPDATE SET target_url = EXCLUDED.target_url, enabled = EXCLUDED.enabled;


-- Seed skill taxonomy
INSERT INTO skills (name, category) VALUES
    ('Go', 'Language'), ('Python', 'Language'), ('TypeScript', 'Language'),
    ('Rust', 'Language'), ('JavaScript', 'Language'), ('SQL', 'Language'),
    ('PostgreSQL', 'Database'), ('MySQL', 'Database'), ('Redis', 'Database'),
    ('MongoDB', 'Database'), ('Elasticsearch', 'Database'),
    ('Docker', 'DevOps'), ('Kubernetes', 'DevOps'), ('CI/CD', 'DevOps'),
    ('AWS', 'Cloud'), ('GCP', 'Cloud'), ('Azure', 'Cloud'),
    ('Terraform', 'DevOps'), ('Helm', 'DevOps'), ('GitHub Actions', 'DevOps'),
    ('REST API', 'Architecture'), ('gRPC', 'Architecture'), ('GraphQL', 'Architecture'),
    ('Microservices', 'Architecture'), ('Linux', 'Tool'), ('Git', 'Tool'),
    ('Gin', 'Framework'), ('React', 'Framework'), ('Node.js', 'Framework')
ON CONFLICT (name) DO NOTHING;
`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}
	log.Println("[DB] Migrations completed successfully")
	return nil
}
