import React, { useState, useEffect, useCallback } from 'react';
import Navbar from './components/layout/Navbar';
import StatsOverview from './components/dashboard/StatsOverview';
import JobFilterBar from './components/dashboard/JobFilterBar';
import JobCard from './components/dashboard/JobCard';
import JobDetailModal from './components/job-detail/JobDetailModal';
import ResumeUploader from './components/resume/ResumeUploader';
import SourceManager from './components/settings/SourceManager';
import { useJobs } from './hooks/useJobs';
import { useResume } from './hooks/useResume';
import { checkHealth } from './services/api';
import { Job, JobFilterParams } from './types';
import { Loader2, SearchX, Wifi, WifiOff } from 'lucide-react';

type ActiveTab = 'dashboard' | 'resume' | 'settings';

function App() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('dashboard');
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [currentFilters, setCurrentFilters] = useState<JobFilterParams>({ days: 1, page: 1, limit: 20 });
  const [apiHealthy, setApiHealthy] = useState(false);

  const { jobs, total, loading, error, refresh, triggerManualScrape, scraping } = useJobs(currentFilters);
  const { resume } = useResume();

  // Health check on mount
  useEffect(() => {
    checkHealth().then(setApiHealthy);
    const interval = setInterval(() => checkHealth().then(setApiHealthy), 30000);
    return () => clearInterval(interval);
  }, []);

  const handleFilter = useCallback((params: JobFilterParams) => {
    setCurrentFilters(params);
    refresh(params);
  }, [refresh]);

  const highAtsCount = jobs.filter(j => j.ats_score !== null && j.ats_score >= 80).length;

  return (
    <div className="min-h-screen bg-surface">
      {/* Ambient glow effects */}
      <div className="fixed inset-0 pointer-events-none overflow-hidden">
        <div className="absolute -top-40 -right-40 w-96 h-96 bg-brand-600/10 rounded-full blur-3xl" />
        <div className="absolute top-1/2 -left-40 w-96 h-96 bg-purple-600/5 rounded-full blur-3xl" />
      </div>

      <Navbar
        activeTab={activeTab}
        onTabChange={setActiveTab}
        onScrape={triggerManualScrape}
        scraping={scraping}
        jobCount={total}
        apiHealthy={apiHealthy}
      />

      {/* API status banner */}
      {!apiHealthy && (
        <div className="bg-amber-500/10 border-b border-amber-500/20 px-4 py-2 flex items-center justify-center gap-2">
          <WifiOff className="w-3.5 h-3.5 text-amber-400" />
          <span className="text-xs text-amber-300">Backend API offline. Start the Go server on port 8080.</span>
        </div>
      )}

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 relative">
        {/* ─── Dashboard Tab ─── */}
        {activeTab === 'dashboard' && (
          <div>
            <StatsOverview
              total={total}
              highAts={highAtsCount}
              skillCount={resume?.extracted_skills?.length ?? 0}
              lastScrape={jobs[0]?.scraped_at}
            />

            <JobFilterBar onFilter={handleFilter} loading={loading} />

            {/* Job grid */}
            {loading && (
              <div className="flex flex-col items-center py-20 gap-4">
                <Loader2 className="w-8 h-8 text-brand-400 animate-spin" />
                <p className="text-sm text-gray-400">Loading jobs...</p>
              </div>
            )}

            {!loading && error && (
              <div className="flex flex-col items-center py-20 gap-3 text-center">
                <WifiOff className="w-10 h-10 text-gray-600" />
                <p className="text-sm text-gray-400">Unable to reach the backend API</p>
                <p className="text-xs text-gray-600">Make sure the Go server is running on port 8080</p>
                <button onClick={() => refresh(currentFilters)} className="btn-primary text-xs mt-2">
                  Retry
                </button>
              </div>
            )}

            {!loading && !error && jobs.length === 0 && (
              <div className="flex flex-col items-center py-20 gap-3 text-center">
                <SearchX className="w-10 h-10 text-gray-600" />
                <p className="text-sm text-gray-400">No jobs found matching your filters</p>
                <p className="text-xs text-gray-600">Try clicking "Scrape Now" to fetch fresh job listings</p>
              </div>
            )}

            {!loading && jobs.length > 0 && (
              <>
                <div className="flex items-center justify-between mb-4">
                  <p className="text-sm text-gray-400">
                    Showing <span className="text-white font-medium">{jobs.length}</span> of{' '}
                    <span className="text-white font-medium">{total}</span> jobs
                  </p>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                  {jobs.map(job => (
                    <JobCard key={job.id} job={job} onClick={setSelectedJob} />
                  ))}
                </div>
              </>
            )}
          </div>
        )}

        {/* ─── Resume Tab ─── */}
        {activeTab === 'resume' && (
          <div className="max-w-2xl mx-auto">
            <div className="mb-6">
              <h2 className="text-xl font-bold text-white">Resume Manager</h2>
              <p className="text-sm text-gray-400 mt-1">
                Upload your resume to enable AI-powered ATS matching and skill gap analysis.
              </p>
            </div>
            <ResumeUploader />
          </div>
        )}

        {/* ─── Settings Tab ─── */}
        {activeTab === 'settings' && (
          <div className="max-w-3xl mx-auto">
            <div className="mb-6">
              <h2 className="text-xl font-bold text-white">Settings</h2>
              <p className="text-sm text-gray-400 mt-1">
                Configure job board scraping targets and scheduling preferences.
              </p>
            </div>
            <div className="glass rounded-xl p-6">
              <SourceManager />
            </div>
          </div>
        )}
      </main>

      {/* Job Detail Modal */}
      <JobDetailModal job={selectedJob} onClose={() => setSelectedJob(null)} />
    </div>
  );
}

export default App;
