import { useState, useEffect } from 'react';
import { getActiveResume, uploadResume } from '../services/api';
import { Resume } from '../types';

export const useResume = () => {
  const [resume, setResume] = useState<Resume | null>(null);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchActive = async () => {
    setLoading(true);
    try {
      const r = await getActiveResume();
      setResume(r);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchActive();
  }, []);

  const upload = async (file: File) => {
    setUploading(true);
    setError(null);
    try {
      const newResumeData = await uploadResume(file);
      const activeRes = await getActiveResume();
      setResume(activeRes);
      return activeRes;
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Upload failed');
      throw err;
    } finally {
      setUploading(false);
    }
  };

  return { resume, loading, uploading, error, upload, refresh: fetchActive };
};
