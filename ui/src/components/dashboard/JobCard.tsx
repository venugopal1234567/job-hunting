import React from 'react';
import { ExternalLink, MapPin, DollarSign, Cpu, ChevronRight, Calendar } from 'lucide-react';
import { Job } from '../../types';

interface JobCardProps {
  job: Job;
  onClick: (job: Job) => void;
}

const SOURCE_COLORS: Record<string, string> = {
  golangprojects: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30',
  hnhiring: 'bg-orange-500/20 text-orange-300 border-orange-500/30',
  weworkremotely: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
  builtin: 'bg-green-500/20 text-green-300 border-green-500/30',
  welcometothejungle: 'bg-pink-500/20 text-pink-300 border-pink-500/30',
};

const ATS_COLOR = (score: number) => {
  if (score >= 80) return { ring: 'stroke-emerald-400', text: 'text-emerald-400', bg: 'bg-emerald-400' };
  if (score >= 60) return { ring: 'stroke-amber-400', text: 'text-amber-400', bg: 'bg-amber-400' };
  return { ring: 'stroke-red-400', text: 'text-red-400', bg: 'bg-red-400' };
};

const JobCard: React.FC<JobCardProps> = ({ job, onClick }) => {
  const hasAts = job.ats_score !== null && job.ats_score !== undefined;
  const atsColor = hasAts ? ATS_COLOR(job.ats_score!) : null;
  const sourceColorClass = SOURCE_COLORS[job.source_board] || 'bg-gray-500/20 text-gray-300 border-gray-500/30';

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return null;
    const d = new Date(dateStr);
    const diff = Math.floor((Date.now() - d.getTime()) / 86400000);
    if (diff === 0) return 'Today';
    if (diff === 1) return 'Yesterday';
    return `${diff}d ago`;
  };

  const circumference = 2 * Math.PI * 15; // radius 15
  const offset = hasAts ? circumference - (job.ats_score! / 100) * circumference : circumference;

  return (
    <div
      id={`job-card-${job.id}`}
      className="glass-hover rounded-xl p-5 cursor-pointer group animate-slide-up"
      onClick={() => onClick(job)}
    >
      <div className="flex items-start justify-between gap-3">
        {/* Company avatar placeholder */}
        <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-surface-300 to-surface-400 flex items-center justify-center flex-shrink-0 text-xs font-bold text-gray-400 border border-white/5">
          {job.company.charAt(0).toUpperCase()}
        </div>

        {/* Job info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <h3 className="text-sm font-semibold text-white group-hover:text-brand-300 transition-colors leading-snug line-clamp-2">
              {job.title}
            </h3>
            <ChevronRight className="w-4 h-4 text-gray-600 group-hover:text-brand-400 flex-shrink-0 mt-0.5 transition-colors" />
          </div>
          <p className="text-xs text-gray-400 mt-0.5">{job.company}</p>
        </div>

        {/* ATS Score ring */}
        {hasAts && atsColor && (
          <div className="flex-shrink-0 flex flex-col items-center">
            <svg width="42" height="42" className="-rotate-90">
              <circle cx="21" cy="21" r="15" fill="none" stroke="rgba(255,255,255,0.06)" strokeWidth="3" />
              <circle
                cx="21" cy="21" r="15"
                fill="none"
                className={`${atsColor.ring} transition-all duration-700`}
                strokeWidth="3"
                strokeDasharray={circumference}
                strokeDashoffset={offset}
                strokeLinecap="round"
              />
            </svg>
            <span className={`text-[10px] font-bold ${atsColor.text} -mt-1`}>{job.ats_score}%</span>
          </div>
        )}
      </div>

      {/* Meta row */}
      <div className="flex flex-wrap items-center gap-2 mt-3">
        {job.location && (
          <span className="flex items-center gap-1 text-[11px] text-gray-500">
            <MapPin className="w-3 h-3" />
            {job.location}
          </span>
        )}
        {job.salary_range && (
          <span className="flex items-center gap-1 text-[11px] text-gray-500">
            <DollarSign className="w-3 h-3" />
            {job.salary_range}
          </span>
        )}
        <span className={`border text-[10px] font-medium px-1.5 py-0.5 rounded-full ${sourceColorClass}`}>
          {job.source_board}
        </span>
        {job.posted_at && (
          <span className="flex items-center gap-1 text-[11px] text-gray-600 ml-auto">
            <Calendar className="w-3 h-3" />
            {formatDate(job.posted_at)}
          </span>
        )}
      </div>

      {/* Card Footer with Skills & Direct Apply Link */}
      <div className="flex flex-wrap items-center justify-between gap-2 mt-3 pt-3 border-t border-white/5">
        <div className="flex flex-wrap gap-1.5 flex-1 min-w-0">
          {job.matched_skills?.slice(0, 3).map(skill => (
            <span key={skill} className="skill-pill-green">{skill}</span>
          ))}
          {job.missing_skills?.slice(0, 1).map(skill => (
            <span key={skill} className="skill-pill-amber">{skill}</span>
          ))}
        </div>
        <a
          href={job.source_url}
          target="_blank"
          rel="noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="text-[11px] font-semibold text-brand-300 hover:text-white bg-brand-600/20 hover:bg-brand-600/50 border border-brand-500/30 px-2.5 py-1 rounded-lg transition-all flex items-center gap-1 flex-shrink-0"
        >
          Apply <ExternalLink className="w-3 h-3" />
        </a>
      </div>
    </div>
  );
};

export default JobCard;
