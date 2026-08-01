import React from 'react';
import { Search, Zap, Globe, FileText, Settings, RefreshCw } from 'lucide-react';

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
    <nav className="glass border-b border-white/5 sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <div className="flex items-center gap-3">
            <div className="relative w-9 h-9 rounded-xl bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center shadow-lg shadow-brand-500/20">
              <Search className="w-4 h-4 text-white" />
              <span className="absolute -top-1 -right-1 w-3 h-3 bg-emerald-400 rounded-full border-2 border-surface" title={apiHealthy ? 'API Online' : 'API Offline'} />
            </div>
            <div>
              <h1 className="text-sm font-bold text-white leading-tight">RemoteHunter</h1>
              <p className="text-[10px] text-gray-500 leading-tight">AI-Powered Job Search</p>
            </div>
          </div>

          {/* Navigation tabs */}
          <div className="flex items-center gap-1 bg-surface-200 rounded-xl p-1">
            <button
              id="nav-dashboard"
              onClick={() => onTabChange('dashboard')}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                activeTab === 'dashboard'
                  ? 'bg-surface-300 text-white shadow-sm'
                  : 'text-gray-400 hover:text-gray-200'
              }`}
            >
              <Globe className="w-3.5 h-3.5" />
              Jobs
              {jobCount > 0 && (
                <span className="bg-brand-600/80 text-brand-100 text-[10px] font-bold px-1.5 py-0.5 rounded-full">
                  {jobCount}
                </span>
              )}
            </button>
            <button
              id="nav-resume"
              onClick={() => onTabChange('resume')}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                activeTab === 'resume'
                  ? 'bg-surface-300 text-white shadow-sm'
                  : 'text-gray-400 hover:text-gray-200'
              }`}
            >
              <FileText className="w-3.5 h-3.5" />
              Resume
            </button>
            <button
              id="nav-settings"
              onClick={() => onTabChange('settings')}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                activeTab === 'settings'
                  ? 'bg-surface-300 text-white shadow-sm'
                  : 'text-gray-400 hover:text-gray-200'
              }`}
            >
              <Settings className="w-3.5 h-3.5" />
              Settings
            </button>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-3">
            <button
              id="btn-trigger-scrape"
              onClick={onScrape}
              disabled={scraping}
              className="btn-primary flex items-center gap-2 text-sm"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${scraping ? 'animate-spin' : ''}`} />
              {scraping ? 'Scraping...' : 'Scrape Now'}
            </button>
            <div className="flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              <span className="text-xs text-gray-400">Gemma 4</span>
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
