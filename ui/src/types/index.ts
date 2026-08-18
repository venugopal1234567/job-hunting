export interface Job {
  id: string;
  title: string;
  company: string;
  location: string;
  country: string;
  source_url: string;
  source_board: string;
  description: string;
  salary_range: string;
  job_type: string;
  posted_at: string | null;
  scraped_at: string;
  is_active: boolean;
  matched_skills: string[];
  missing_skills: string[];
  ats_score: number | null;
}

export interface JobsResponse {
  total: number;
  page: number;
  limit: number;
  jobs: Job[];
}

export interface JobFilterParams {
  skills?: string;
  days?: number;
  country?: string;
  only_enabled?: boolean;
  sources?: string;
  page?: number;
  limit?: number;
}

export interface Resume {
  id: string;
  filename: string;
  extracted_skills: string[];
  raw_text_length: number;
  uploaded_at: string;
  is_active: boolean;
  has_pdf?: boolean;
}

export interface GapQuestion {
  skill: string;
  question: string;
}

export interface MatchBreakdown {
  matched_skills: string[];
  missing_skills: string[];
}

export interface ATSAnalysis {
  id: string;
  job_id: string;
  ats_score: number;
  match_breakdown: MatchBreakdown;
  actionable_suggestions: string[];
  gap_questions: GapQuestion[];
  analyzed_at: string;
}

export interface ScraperConfig {
  id: number;
  board_name: string;
  target_url: string;
  enabled: boolean;
  cron_schedule: string;
  last_run_at: string | null;
}

export interface Settings {
  sources: ScraperConfig[];
  default_skills: string[];
  scrape_interval: string;
}

// ─── Resume Editor Types ──────────────────────────────────────────────────────

export interface ProposedEdit {
  id: string;
  original: string;
  replacement: string;
  reason: string;
}

export interface GapQuestionPrompt {
  skill: string;
  question: string;
}

export type ChatMessageRole = 'user' | 'assistant' | 'system';

export interface SkillCategory {
  category: string;
  items: string;
}

export interface JobExperience {
  title: string;
  date: string;
  company: string;
  location: string;
  bullets: string[];
  tech_stack: string;
}

export interface EducationItem {
  institution: string;
  date: string;
  degree: string;
}

export interface StructuredResume {
  name: string;
  title: string;
  contact_items: string[];
  summary: string;
  skills: SkillCategory[];
  work_experience: JobExperience[];
  education: EducationItem[];
  highlight_keywords: string[];
}

export interface ChatMessage {
  id: string;
  role: ChatMessageRole;
  content: string;
  proposedEdits?: ProposedEdit[];
  gapPrompts?: GapQuestionPrompt[];
  structuredResume?: StructuredResume;
  timestamp: number;
}

export type TrackedChangeStatus = 'pending' | 'accepted' | 'rejected';

export interface TrackedChange {
  id: string;
  original: string;
  replacement: string;
  reason: string;
  status: TrackedChangeStatus;
  messageId: string;
}

export interface ResumeVersion {
  id: string;
  resume_id: string;
  job_id?: string;
  job_title?: string;
  job_company?: string;
  label: string;
  applied_at: string;
  source: 'editor' | 'upload';
  snapshot_structured?: StructuredResume;
}
