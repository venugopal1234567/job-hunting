import React, { useState, KeyboardEvent } from 'react';
import { Search, X, SlidersHorizontal } from 'lucide-react';
import { JobFilterParams } from '../../types';

interface JobFilterBarProps {
  onFilter: (params: JobFilterParams) => void;
  loading: boolean;
}

const DAYS_OPTIONS = [
  { label: '24 Hours', value: 1 },
  { label: '3 Days', value: 3 },
  { label: '7 Days', value: 7 },
  { label: '30 Days', value: 30 },
];

const SKILL_SUGGESTIONS = [
  'Go', 'Python', 'TypeScript', 'Rust', 'Docker', 'Kubernetes',
  'PostgreSQL', 'AWS', 'GCP', 'gRPC', 'React', 'Redis', 'Kafka',
];

const JobFilterBar: React.FC<JobFilterBarProps> = ({ onFilter, loading }) => {
  // Load initially from localStorage if present
  const [selectedSkills, setSelectedSkills] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem('filter_skills');
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });
  const [skillInput, setSkillInput] = useState('');
  const [days, setDays] = useState<number>(() => {
    const saved = localStorage.getItem('filter_days');
    return saved ? parseInt(saved, 10) : 30;
  });
  const [countries, setCountries] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem('filter_countries');
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });
  const [onlyEnabled, setOnlyEnabled] = useState<boolean>(() => {
    const saved = localStorage.getItem('filter_only_enabled');
    return saved ? saved === 'true' : true;
  });
  const [selectedSources, setSelectedSources] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem('filter_sources');
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });

  // Apply filters on mount
  React.useEffect(() => {
    applyFilter(selectedSkills, days, countries, onlyEnabled, selectedSources);
  }, []);

  const addSkill = (skill: string) => {
    const s = skill.trim();
    if (s && !selectedSkills.includes(s)) {
      const updated = [...selectedSkills, s];
      setSelectedSkills(updated);
      localStorage.setItem('filter_skills', JSON.stringify(updated));
      applyFilter(updated, days, countries, onlyEnabled, selectedSources);
    }
    setSkillInput('');
  };

  const removeSkill = (skill: string) => {
    const updated = selectedSkills.filter(s => s !== skill);
    setSelectedSkills(updated);
    localStorage.setItem('filter_skills', JSON.stringify(updated));
    applyFilter(updated, days, countries, onlyEnabled, selectedSources);
  };

  const applyFilter = (skills: string[], d: number, cList: string[], enabledOnly: boolean, boards: string[]) => {
    onFilter({
      skills: skills.length > 0 ? skills.join(',') : undefined,
      days: d,
      country: cList.length > 0 ? cList.join(',') : undefined,
      only_enabled: enabledOnly,
      sources: boards.length > 0 ? boards.join(',') : undefined,
      page: 1,
      limit: 20,
    });
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if ((e.key === 'Enter' || e.key === ',') && skillInput) {
      e.preventDefault();
      addSkill(skillInput);
    }
  };

  const handleDaysChange = (d: number) => {
    setDays(d);
    localStorage.setItem('filter_days', d.toString());
    applyFilter(selectedSkills, d, countries, onlyEnabled, selectedSources);
  };

  const toggleCountry = (c: string) => {
    let updated: string[];
    if (countries.includes(c)) {
      updated = countries.filter(item => item !== c);
    } else {
      updated = [...countries, c];
    }
    setCountries(updated);
    localStorage.setItem('filter_countries', JSON.stringify(updated));
    applyFilter(selectedSkills, days, updated, onlyEnabled, selectedSources);
  };

  const toggleSource = (board: string) => {
    let updated: string[];
    if (selectedSources.includes(board)) {
      updated = selectedSources.filter(item => item !== board);
    } else {
      updated = [...selectedSources, board];
    }
    setSelectedSources(updated);
    localStorage.setItem('filter_sources', JSON.stringify(updated));
    applyFilter(selectedSkills, days, countries, onlyEnabled, updated);
  };

  const handleOnlyEnabledChange = (checked: boolean) => {
    setOnlyEnabled(checked);
    localStorage.setItem('filter_only_enabled', checked.toString());
    applyFilter(selectedSkills, days, countries, checked, selectedSources);
  };

  const clearAll = () => {
    setSelectedSkills([]);
    setCountries([]);
    setDays(30);
    setOnlyEnabled(true);
    setSelectedSources([]);
    localStorage.removeItem('filter_skills');
    localStorage.removeItem('filter_countries');
    localStorage.setItem('filter_days', '30');
    localStorage.setItem('filter_only_enabled', 'true');
    localStorage.removeItem('filter_sources');
    applyFilter([], 30, [], true, []);
  };

  const hasFilters = selectedSkills.length > 0 || countries.length > 0 || days !== 30 || !onlyEnabled || selectedSources.length > 0;

  return (
    <div className="glass rounded-xl p-4 mb-5 space-y-4">
      <div className="flex items-center gap-2 mb-1">
        <SlidersHorizontal className="w-4 h-4 text-brand-400" />
        <span className="text-sm font-semibold text-white">Filter Jobs</span>
        {hasFilters && (
          <button onClick={clearAll} className="ml-auto text-xs text-gray-500 hover:text-red-400 transition-colors">
            Clear all
          </button>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Skills Multi-Select */}
        <div className="md:col-span-1">
          <label className="section-header">Target Skills</label>
          <div className="flex flex-wrap gap-1.5 mb-2">
            {selectedSkills.map(skill => (
              <span key={skill} className="skill-pill-blue flex items-center gap-1">
                {skill}
                <button onClick={() => removeSkill(skill)} className="hover:text-white ml-0.5">
                  <X className="w-2.5 h-2.5" />
                </button>
              </span>
            ))}
          </div>
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-500" />
            <input
              id="skill-filter-input"
              type="text"
              value={skillInput}
              onChange={e => setSkillInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Add skill, press Enter..."
              className="input-field w-full pl-8"
            />
          </div>
          {/* Quick suggestions */}
          <div className="flex flex-wrap gap-1 mt-2">
            {SKILL_SUGGESTIONS.filter(s => !selectedSkills.includes(s)).slice(0, 5).map(skill => (
              <button
                key={skill}
                onClick={() => addSkill(skill)}
                className="text-[10px] text-gray-500 hover:text-brand-400 bg-surface-200 hover:bg-surface-300 px-1.5 py-0.5 rounded transition-colors"
              >
                +{skill}
              </button>
            ))}
          </div>
        </div>

        {/* Days Filter */}
        <div>
          <label className="section-header">Posted Within</label>
          <div className="grid grid-cols-2 gap-1.5">
            {DAYS_OPTIONS.map(opt => (
              <button
                key={opt.value}
                id={`days-filter-${opt.value}`}
                onClick={() => handleDaysChange(opt.value)}
                className={`py-1.5 rounded-lg text-xs font-medium transition-all duration-150 ${
                  days === opt.value
                    ? 'bg-brand-600 text-white shadow-sm shadow-brand-500/30'
                    : 'bg-surface-200 text-gray-400 hover:bg-surface-300 hover:text-white'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        {/* Country Filter */}
        <div>
          <label className="section-header">Location / Country</label>
          <div className="flex flex-col gap-1.5">
            <input
              id="country-filter-input"
              type="text"
              value={countries.join(', ')}
              onChange={e => {
                const parts = e.target.value.split(',').map(s => s.trim()).filter(Boolean);
                setCountries(parts);
                localStorage.setItem('filter_countries', JSON.stringify(parts));
                applyFilter(selectedSkills, days, parts, onlyEnabled, selectedSources);
              }}
              placeholder="e.g. Worldwide, US, India (comma separated)..."
              className="input-field w-full"
            />
            <div className="flex gap-1">
              {['Worldwide', 'US', 'EU', 'India'].map(c => (
                <button
                  key={c}
                  onClick={() => toggleCountry(c)}
                  className={`flex-1 py-1 rounded text-[10px] font-medium transition-colors ${
                    countries.includes(c)
                      ? 'bg-brand-600/80 text-white'
                      : 'bg-surface-200 text-gray-500 hover:text-gray-300'
                  }`}
                >
                  {c}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Sources Filter */}
        <div>
          <label className="section-header">Scraped Sources</label>
          <div className="flex flex-wrap gap-1 mt-1 mb-2">
            {[
              { label: 'FlexBoard', value: 'flexboard' },
              { label: 'BuiltIn', value: 'builtin' },
              { label: 'VacancyPro', value: 'vacancyglobalpro' },
              { label: 'RemoteOK', value: 'remoteok' },
              { label: 'WWR', value: 'weworkremotely' },
              { label: 'Remotive', value: 'remotive' },
              { label: 'Arbeitnow', value: 'arbeitnow' },
              { label: 'GolangProj', value: 'golangprojects' },
              { label: 'HNHiring', value: 'hnhiring' },
              { label: 'GoogleJobs', value: 'googlejobs' },
            ].map(src => (
              <button
                key={src.value}
                onClick={() => toggleSource(src.value)}
                className={`py-1 px-2 rounded text-[10px] font-medium transition-colors ${
                  selectedSources.includes(src.value)
                    ? 'bg-brand-600/80 text-white'
                    : 'bg-surface-200 text-gray-500 hover:text-gray-300'
                }`}
              >
                {src.label}
              </button>
            ))}
          </div>
          <div className="flex items-center space-x-2 mt-2">
            <input
              type="checkbox"
              id="only-enabled-checkbox"
              checked={onlyEnabled}
              onChange={e => handleOnlyEnabledChange(e.target.checked)}
              className="w-4 h-4 accent-brand-600 rounded bg-surface-200 border-gray-600 focus:ring-brand-500 focus:ring-offset-gray-900 cursor-pointer"
            />
            <label htmlFor="only-enabled-checkbox" className="text-xs text-gray-300 font-medium cursor-pointer select-none">
              Only Enabled Sources
            </label>
          </div>
        </div>
      </div>
    </div>
  );
};

export default JobFilterBar;
