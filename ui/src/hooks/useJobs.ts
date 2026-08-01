import { useState, useCallback, useEffect } from 'react';
import { getJobs, triggerScrape } from '../services/api';
import { Job, JobFilterParams, JobsResponse } from '../types';

interface UseJobsState {
  jobs: Job[];
  total: number;
  page: number;
  loading: boolean;
  error: string | null;
  scraping: boolean;
}

export const useJobs = (filters: JobFilterParams = {}) => {
  const [state, setState] = useState<UseJobsState>({
    jobs: [],
    total: 0,
    page: 1,
    loading: true,
    error: null,
    scraping: false,
  });

  const fetchJobs = useCallback(async (params: JobFilterParams = {}) => {
    setState(prev => ({ ...prev, loading: true, error: null }));
    try {
      const res: JobsResponse = await getJobs(params);
      setState(prev => ({
        ...prev,
        jobs: res.jobs || [],
        total: res.total,
        page: res.page,
        loading: false,
      }));
    } catch (err: any) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: err.message || 'Failed to load jobs',
      }));
    }
  }, []);

  useEffect(() => {
    fetchJobs(filters);
  }, []);

  const refresh = (params?: JobFilterParams) => fetchJobs(params ?? filters);

  const triggerManualScrape = async () => {
    setState(prev => ({ ...prev, scraping: true }));
    try {
      await triggerScrape();
      setTimeout(() => {
        fetchJobs(filters);
        setState(prev => ({ ...prev, scraping: false }));
      }, 3000);
    } catch {
      setState(prev => ({ ...prev, scraping: false }));
    }
  };

  return { ...state, refresh, triggerManualScrape };
};
