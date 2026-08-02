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
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [skillInput, setSkillInput] = useState('');
  const [days, setDays] = useState(30);
  const [country, setCountry] = useState('');

  const addSkill = (skill: string) => {
    const s = skill.trim();
    if (s && !selectedSkills.includes(s)) {
      const updated = [...selectedSkills, s];
      setSelectedSkills(updated);
      applyFilter(updated, days, country);
    }
    setSkillInput('');
  };

  const removeSkill = (skill: string) => {
    const updated = selectedSkills.filter(s => s !== skill);
    setSelectedSkills(updated);
    applyFilter(updated, days, country);
  };

  const applyFilter = (skills: string[], d: number, c: string) => {
    onFilter({
      skills: skills.length > 0 ? skills.join(',') : undefined,
      days: d,
      country: c || undefined,
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
    applyFilter(selectedSkills, d, country);
  };

  const handleCountryChange = (c: string) => {
    setCountry(c);
    applyFilter(selectedSkills, days, c);
  };

  const clearAll = () => {
    setSelectedSkills([]);
    setCountry('');
    setDays(30);
    applyFilter([], 30, '');
  };

  const hasFilters = selectedSkills.length > 0 || country || days !== 30;

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

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
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
              value={country}
              onChange={e => handleCountryChange(e.target.value)}
              placeholder="e.g. Worldwide, US, India..."
              className="input-field w-full"
            />
            <div className="flex gap-1">
              {['Worldwide', 'US', 'EU', 'India'].map(c => (
                <button
                  key={c}
                  onClick={() => handleCountryChange(c === country ? '' : c)}
                  className={`flex-1 py-1 rounded text-[10px] font-medium transition-colors ${
                    country === c
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
      </div>
    </div>
  );
};

export default JobFilterBar;
