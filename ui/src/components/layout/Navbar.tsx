import React from 'react';
import { Search, Sparkles, Globe, FileText, Settings, RefreshCw, Radio } from 'lucide-react';

interface NavbarProps {
  activeTab: 'dashboard' | 'resume' | 'settings';
  onTabChange: (tab: 'dashboard' | 'resume' | 'settings') => void;
  onScrape: () => void;
  scraping: boolean;
  jobCount: number;
  apiHealthy: boolean;
}

const Navbar: React.FC<NavbarProps> = ({ activeTab, onTabChange, onScrape, scraping, jobCount, apiHealthy }) => {
  return (
    <nav className="bg-surface-container-low/90 backdrop-blur-xl rounded-full mt-4 mx-auto w-[95%] max-w-7xl sticky top-4 shadow-elevation-3 flex justify-between items-center px-4 sm:px-6 py-3 z-50 border border-surface-container-highest transition-all">
      {/* Brand */}
      <div className="flex items-center gap-3">
        <div className="relative w-10 h-10 rounded-full bg-primary flex items-center justify-center text-on-primary shadow-elevation-1">
          <Radio className="w-5 h-5" />
          <span
            className={`absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-white ${
              apiHealthy ? 'bg-emerald-500' : 'bg-red-500'
            }`}
            title={apiHealthy ? 'API Online' : 'API Offline'}
          />
        </div>
        <div className="hidden sm:block">
          <h1 className="text-base font-bold text-primary leading-tight font-headline">RemoteHunter</h1>
          <p className="text-[10px] text-on-surface-variant leading-tight">Material AI Job Search</p>
        </div>
      </div>

      {/* Navigation tabs */}
      <div className="flex items-center gap-1.5 bg-surface-container p-1 rounded-full">
        <button
          id="nav-dashboard"
          onClick={() => onTabChange('dashboard')}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold transition-all duration-200 ${
            activeTab === 'dashboard'
              ? 'bg-primary text-on-primary shadow-sm'
              : 'text-on-surface-variant hover:text-primary hover:bg-surface-container-high'
          }`}
        >
          <Globe className="w-3.5 h-3.5" />
          <span>Jobs</span>
          {jobCount > 0 && (
            <span
              className={`text-[10px] font-bold px-1.5 py-0.2 rounded-full ${
                activeTab === 'dashboard'
                  ? 'bg-on-primary text-primary'
                  : 'bg-primary-fixed text-on-primary-fixed'
              }`}
            >
              {jobCount}
            </span>
          )}
        </button>

        <button
          id="nav-resume"
          onClick={() => onTabChange('resume')}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold transition-all duration-200 ${
            activeTab === 'resume'
              ? 'bg-primary text-on-primary shadow-sm'
              : 'text-on-surface-variant hover:text-primary hover:bg-surface-container-high'
          }`}
        >
          <FileText className="w-3.5 h-3.5" />
          <span>Resume</span>
        </button>

        <button
          id="nav-settings"
          onClick={() => onTabChange('settings')}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold transition-all duration-200 ${
            activeTab === 'settings'
              ? 'bg-primary text-on-primary shadow-sm'
              : 'text-on-surface-variant hover:text-primary hover:bg-surface-container-high'
          }`}
        >
          <Settings className="w-3.5 h-3.5" />
          <span>Settings</span>
        </button>
      </div>

      {/* Trailing Actions */}
      <div className="flex items-center gap-2.5">
        <button
          id="btn-trigger-scrape"
          onClick={onScrape}
          disabled={scraping}
          className="btn-primary text-xs py-2 px-4 flex items-center gap-2"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${scraping ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline">{scraping ? 'Scraping...' : 'Scrape Now'}</span>
        </button>

        <div className="hidden md:flex items-center gap-1.5 bg-primary-fixed text-on-primary-fixed px-3 py-1.5 rounded-full text-xs font-medium border border-primary-fixed-dim/50">
          <Sparkles className="w-3.5 h-3.5 text-primary" />
          <span>Gemma 4</span>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;

