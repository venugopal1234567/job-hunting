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
    edited_text TEXT,
    extracted_skills JSONB,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

ALTER TABLE resumes ADD COLUMN IF NOT EXISTS edited_text TEXT;
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS pdf_data BYTEA;

-- 4b. Resume Versions Table (applied snapshots)
CREATE TABLE IF NOT EXISTS resume_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_id UUID REFERENCES resumes(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    snapshot_text TEXT NOT NULL,
    label VARCHAR(255),
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(50) DEFAULT 'editor'
);

CREATE INDEX IF NOT EXISTS idx_resume_versions_resume ON resume_versions(resume_id);
CREATE INDEX IF NOT EXISTS idx_resume_versions_job ON resume_versions(job_id);
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

-- Seed default scraper targets (only inserts if not already present — preserves user edits)
INSERT INTO scraper_configs (board_name, target_url, enabled, cron_schedule)
VALUES
    ('WeWorkRemotely Golang', 'https://weworkremotely.com/remote-jobs-golang', true, '@every 1h'),
    ('RemoteOK', 'https://remoteok.com/api?tag=golang', true, '@every 1h'),
    ('GolangProjects', 'https://www.golangprojects.com/rss.xml', true, '@every 1h'),
    ('BuiltIn Remote', 'https://builtin.com/jobs/remote/senior?search=Go&daysSinceUpdated=7&skills=Go%2CPython%2CAWS%2CDocker%2CGCP%2CTypescript%2CAzure%2CCi%2FCd%2CPostgres%2CRust%2CSQL%2CNode.js&country=IND&allLocations=true', true, '@every 2h'),
    ('FlexBoard', 'https://flexboard.9y.liveblog365.com/?search=golang', true, '@every 1h'),
    ('VacancyGlobalPro', 'https://vacancyglobalpro.up.railway.app/remote-golang-jobs', true, '@every 1h'),
    ('GoogleJobs', 'https://www.google.com/search?q=intitle%3A%22Senior+Golang%22+OR+intitle%3A%22Lead+Go%22+%22remote%22+%22India%22+OR+%22worldwide%22+jobs&udm=8&jbr=sep:0|https://www.google.com/search?q=%22Senior+Golang+Engineer%22+%22100%25+remote%22+OR+%22fully+remote%22+%22worldwide%22+OR+%22Anywhere%22&udm=8&jbr=sep:0', true, '@every 4h'),
    ('RemoteRocketship', 'https://www.remoterocketship.com/?ref=yanirs-established-remote&page=1&sort=DateAdded&jobTitle=Golang&locations=Worldwide%2CIndia', true, '@every 1h'),
    ('LinkedIn', 'https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=golang&location=India&f_WT=2|https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=golang&location=Worldwide&f_WT=2', true, '@every 1h'),
    ('Glassdoor', 'https://www.glassdoor.co.in/Job/india-golang-jobs-SRCH_IL.0,5_KO6,12.htm?remoteWorkType=1', true, '@every 1h'),
    ('Bayt', 'https://www.bayt.com/en/international/jobs/golang-remote-jobs/', true, '@every 1h'),
    ('BDJobs', 'https://jobs.bdjobs.com/jobsearch.asp?txtsearch=golang', true, '@every 1h'),
    ('Indeed', 'https://www.indeed.com/jobs?q=golang&l=Remote', true, '@every 1h'),
    ('Naukri', 'https://www.naukri.com/golang-jobs?wfhType=2', true, '@every 1h'),
    ('ZipRecruiter', 'https://www.ziprecruiter.com/Jobs/Golang?location=Remote', true, '@every 1h'),
    ('RealWorkFromAnywhere', 'https://www.realworkfromanywhere.com/remote-backend-jobs', true, '@every 1h')
ON CONFLICT (board_name) DO NOTHING;

-- 7. App Settings Table (key-value for runtime configuration)
CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed default AI model setting (only if not already set)
INSERT INTO app_settings (key, value)
VALUES ('active_model', 'rafw007/qwen35-claude-coder:4b')
ON CONFLICT (key) DO NOTHING;

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
