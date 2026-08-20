import React, { useState, useEffect, useCallback } from 'react';
import Navbar from './components/layout/Navbar';
import StatsOverview from './components/dashboard/StatsOverview';
import JobFilterBar from './components/dashboard/JobFilterBar';
import JobCard from './components/dashboard/JobCard';
import JobDetailModal from './components/job-detail/JobDetailModal';
import ResumeEditor from './components/resume/ResumeEditor';
import SourceManager from './components/settings/SourceManager';
import AIModelPicker from './components/settings/AIModelPicker';
import { useJobs } from './hooks/useJobs';
import { useResume } from './hooks/useResume';
import { checkHealth } from './services/api';
import { Job, JobFilterParams } from './types';
import { Loader2, SearchX, WifiOff, Sparkles } from 'lucide-react';

type ActiveTab = 'dashboard' | 'resume' | 'settings';

function App() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('dashboard');
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [viewingJob, setViewingJob] = useState<Job | null>(null);
  const [currentFilters, setCurrentFilters] = useState<JobFilterParams>(() => {
    const savedDays = localStorage.getItem('filter_days');
    const days = savedDays ? parseInt(savedDays, 10) : 30;

    let skills: string | undefined;
    try {
      const savedSkills = localStorage.getItem('filter_skills');
      if (savedSkills) {
        const parsed = JSON.parse(savedSkills);
        if (parsed.length > 0) skills = parsed.join(',');
      }
    } catch (_) {}

    let country: string | undefined;
    try {
      const savedCountries = localStorage.getItem('filter_countries');
      if (savedCountries) {
        const parsed = JSON.parse(savedCountries);
        if (parsed.length > 0) country = parsed.join(',');
      }
    } catch (_) {}

    const onlyEnabledStr = localStorage.getItem('filter_only_enabled');
    const only_enabled = onlyEnabledStr ? onlyEnabledStr === 'true' : true;

    let sources: string | undefined;
    try {
      const savedSources = localStorage.getItem('filter_sources');
      if (savedSources) {
        const parsed = JSON.parse(savedSources);
        if (parsed.length > 0) sources = parsed.join(',');
      }
    } catch (_) {}

    return {
      days,
      skills,
      country,
      only_enabled,
      sources,
      page: 1,
      limit: 20
    };
  });
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

  const handleEditResume = useCallback((job: Job) => {
    setSelectedJob(job);   // keep job selected for ATS context
    setViewingJob(null);    // close the detail modal
    setActiveTab('resume');
  }, []);

  const highAtsCount = jobs.filter(j => j.ats_score !== null && j.ats_score >= 80).length;

  return (
    <div className="min-h-screen bg-background text-on-surface pb-16 font-sans">
      {/* Ambient Material 3 glow effects */}
      <div className="fixed inset-0 pointer-events-none overflow-hidden -z-10">
        <div className="absolute -top-40 -right-40 w-96 h-96 bg-primary/5 rounded-full blur-3xl" />
        <div className="absolute top-1/3 -left-40 w-96 h-96 bg-secondary-container/10 rounded-full blur-3xl" />
        <div className="absolute bottom-10 right-1/4 w-96 h-96 bg-tertiary-fixed/15 rounded-full blur-3xl" />
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
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-3">
          <div className="bg-error-container text-on-error-container border border-error/20 rounded-full px-4 py-2 flex items-center justify-center gap-2 shadow-sm text-xs font-medium">
            <WifiOff className="w-3.5 h-3.5" />
            <span>Backend API offline. Start the Go server on port 8080.</span>
          </div>
        </div>
      )}

      <main className={`${activeTab === 'resume' ? 'w-full px-4 lg:px-6' : 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8'} py-6 relative`}>
        {/* ─── Dashboard Tab ─── */}
        {activeTab === 'dashboard' && (
          <div className="space-y-6">
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
                <Loader2 className="w-8 h-8 text-primary animate-spin" />
                <p className="text-sm font-medium text-on-surface-variant">Loading jobs...</p>
              </div>
            )}

            {!loading && error && (
              <div className="bg-surface-container-lowest border border-surface-variant rounded-2xl p-12 text-center flex flex-col items-center gap-3 shadow-elevation-1">
                <div className="p-3 rounded-full bg-error-container text-on-error-container">
                  <WifiOff className="w-8 h-8" />
                </div>
                <p className="text-base font-semibold text-on-surface">Unable to reach the backend API</p>
                <p className="text-xs text-on-surface-variant">Make sure the Go server is running on port 8080</p>
                <button onClick={() => refresh(currentFilters)} className="btn-primary text-xs mt-2">
                  Retry
                </button>
              </div>
            )}

            {!loading && !error && jobs.length === 0 && (
              <div className="bg-surface-container-lowest border border-surface-variant rounded-2xl p-12 text-center flex flex-col items-center gap-3 shadow-elevation-1">
                <div className="p-3 rounded-full bg-surface-container text-on-surface-variant">
                  <SearchX className="w-8 h-8" />
                </div>
                <p className="text-base font-semibold text-on-surface">No jobs found matching your filters</p>
                <p className="text-xs text-on-surface-variant">Try clicking "Scrape Now" to fetch fresh job listings</p>
              </div>
            )}

            {!loading && jobs.length > 0 && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <p className="text-sm text-on-surface-variant">
                    Showing <span className="font-bold text-on-surface">{jobs.length}</span> of{' '}
                    <span className="font-bold text-on-surface">{total}</span> jobs
                  </p>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
                  {jobs.map(job => (
                    <JobCard key={job.id} job={job} onClick={setViewingJob} onEditResume={handleEditResume} />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* ─── Resume Tab ─── */}
        {activeTab === 'resume' && (
          <div className="w-full">
            <ResumeEditor selectedJob={selectedJob} />
          </div>
        )}

        {/* ─── Settings Tab ─── */}
        {activeTab === 'settings' && (
          <div className="max-w-3xl mx-auto">
            <div className="mb-6">
              <h2 className="text-2xl font-bold font-headline text-on-surface">Settings</h2>
              <p className="text-sm text-on-surface-variant mt-1">
                Configure AI models, job board scraping targets, and scheduling preferences.
              </p>
            </div>
            <div className="space-y-6">
              {/* AI Model Picker */}
              <div className="bg-surface-container-lowest border border-surface-variant rounded-2xl p-6 shadow-elevation-1">
                <AIModelPicker />
              </div>
              {/* Job Board Sources */}
              <div className="bg-surface-container-lowest border border-surface-variant rounded-2xl p-6 shadow-elevation-1">
                <SourceManager />
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Job Detail Modal */}
      <JobDetailModal
        job={viewingJob}
        onClose={() => setViewingJob(null)}
        onEditResume={handleEditResume}
      />
    </div>
  );
}

export default App;

