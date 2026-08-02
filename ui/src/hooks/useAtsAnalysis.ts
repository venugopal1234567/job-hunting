import { useState } from 'react';
import { analyzeJob } from '../services/api';
import { ATSAnalysis } from '../types';

export const useAtsAnalysis = () => {
  const [analysis, setAnalysis] = useState<ATSAnalysis | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const analyze = async (jobId: string, resumeId?: string, force = false) => {
    setLoading(true);
    setError(null);
    try {
      const result = await analyzeJob(jobId, resumeId, force);
      setAnalysis(result);
      return result;
    } catch (err: any) {
      const msg = err.response?.data?.error || err.message || 'Analysis failed';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const reset = () => {
    setAnalysis(null);
    setError(null);
  };

  return { analysis, loading, error, analyze, reset };
};
