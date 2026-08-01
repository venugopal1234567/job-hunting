import axios from 'axios';
import { ATSAnalysis, Job, JobFilterParams, JobsResponse, Resume, ScraperConfig, Settings } from '../types';

const BASE_URL = '/api/v1';

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 120_000,
  headers: { 'Content-Type': 'application/json' },
});

// ─── Jobs ──────────────────────────────────────────────────────────────────────

export const getJobs = async (params: JobFilterParams = {}): Promise<JobsResponse> => {
  const { data } = await api.get<JobsResponse>('/jobs', { params });
  return data;
};

export const getJobById = async (id: string): Promise<Job> => {
  const { data } = await api.get<Job>(`/jobs/${id}`);
  return data;
};

export const triggerScrape = async (): Promise<{ message: string }> => {
  const { data } = await api.post('/jobs/trigger-scrape');
  return data;
};

// ─── ATS Analysis ──────────────────────────────────────────────────────────────

export const analyzeJob = async (jobId: string, resumeId?: string): Promise<ATSAnalysis> => {
  const { data } = await api.post<ATSAnalysis>(`/jobs/${jobId}/analyze`, resumeId ? { resume_id: resumeId } : {});
  return data;
};

// ─── Resume ────────────────────────────────────────────────────────────────────

export const uploadResume = async (file: File): Promise<Resume> => {
  const formData = new FormData();
  formData.append('file', file);
  const { data } = await api.post<Resume>('/resume/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return data;
};

export const getActiveResume = async (): Promise<Resume | null> => {
  try {
    const { data } = await api.get<Resume>('/resume/active');
    return data;
  } catch (err: any) {
    if (err.response?.status === 404) return null;
    throw err;
  }
};

// ─── Settings ──────────────────────────────────────────────────────────────────

export const getSettings = async (): Promise<Settings> => {
  const { data } = await api.get<Settings>('/settings');
  return data;
};

export const updateSources = async (sources: ScraperConfig[]): Promise<void> => {
  await api.put('/settings/sources', sources);
};

export const checkHealth = async (): Promise<boolean> => {
  try {
    await api.get('/health');
    return true;
  } catch {
    return false;
  }
};
