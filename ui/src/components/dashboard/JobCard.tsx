import React from 'react';
import { ExternalLink, MapPin, DollarSign, ChevronRight, Calendar, Sparkles } from 'lucide-react';
import { Job } from '../../types';

interface JobCardProps {
  job: Job;
  onClick: (job: Job) => void;
  onEditResume?: (job: Job) => void;
}

const SOURCE_COLORS: Record<string, string> = {
  golangprojects: 'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
  hnhiring: 'bg-secondary-fixed text-on-secondary-fixed border-secondary-fixed-dim/40',
  weworkremotely: 'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
  builtin: 'bg-tertiary-fixed text-on-tertiary-fixed border-tertiary-fixed-dim/40',
  welcometothejungle: 'bg-secondary-fixed text-on-secondary-fixed border-secondary-fixed-dim/40',
  googlejobs: 'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
  remoterocketship: 'bg-tertiary-fixed text-on-tertiary-fixed border-tertiary-fixed-dim/40',
  linkedin: 'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
  glassdoor: 'bg-tertiary-fixed text-on-tertiary-fixed border-tertiary-fixed-dim/40',
  naukri: 'bg-secondary-fixed text-on-secondary-fixed border-secondary-fixed-dim/40',
};

const AVATAR_BG = [
  'bg-primary text-on-primary',
  'bg-secondary text-on-secondary',
  'bg-tertiary text-on-tertiary',
  'bg-primary-container text-on-primary',
  'bg-secondary-container text-on-secondary-container',
  'bg-tertiary-container text-on-tertiary-container',
];

const ATS_COLOR = (score: number) => {
  if (score >= 80) return { ring: 'stroke-primary', text: 'text-primary' };
  if (score >= 60) return { ring: 'stroke-secondary', text: 'text-secondary' };
  return { ring: 'stroke-outline', text: 'text-on-surface-variant' };
};

const JobCard: React.FC<JobCardProps> = ({ job, onClick, onEditResume }) => {
  const hasAts = job.ats_score !== null && job.ats_score !== undefined;
  const atsColor = hasAts ? ATS_COLOR(job.ats_score!) : null;
  const sourceColorClass = SOURCE_COLORS[job.source_board] || 'bg-surface-container text-on-surface border-surface-variant';

  // Deterministic avatar color based on company name
  const avatarIndex = (job.company || 'A').charCodeAt(0) % AVATAR_BG.length;
  const avatarBgClass = AVATAR_BG[avatarIndex];

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
      className="bg-surface-container-lowest border border-surface-variant rounded-2xl p-5 sm:p-6 cursor-pointer group shadow-elevation-1 hover:shadow-elevation-2 hover:border-primary/40 transition-all duration-200 flex flex-col justify-between animate-slide-up"
      onClick={() => onClick(job)}
    >
      <div>
        {/* Card Header with Avatar & Title */}
        <div className="flex items-start justify-between gap-3.5">
          {/* Company letter avatar */}
          <div className={`w-11 h-11 rounded-full ${avatarBgClass} flex items-center justify-center flex-shrink-0 text-sm font-bold shadow-sm`}>
            {job.company.charAt(0).toUpperCase()}
          </div>

          {/* Job info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <h3 className="text-base font-bold font-headline text-on-surface group-hover:text-primary transition-colors leading-snug line-clamp-2">
                {job.title}
              </h3>
              <ChevronRight className="w-4 h-4 text-outline group-hover:text-primary flex-shrink-0 mt-1 transition-colors" />
            </div>
            <p className="text-xs font-medium text-on-surface-variant mt-0.5">{job.company}</p>
          </div>

          {/* ATS Score ring */}
          {hasAts && atsColor && (
            <div className="flex-shrink-0 flex flex-col items-center">
              <svg width="42" height="42" className="-rotate-90">
                <circle cx="21" cy="21" r="15" fill="none" stroke="#ededf2" strokeWidth="3" />
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

        {/* Meta badges row */}
        <div className="flex flex-wrap items-center gap-2 mt-4">
          {job.location && (
            <span className="bg-surface-container text-on-surface px-2.5 py-1 rounded-full text-[11px] font-medium flex items-center gap-1">
              <MapPin className="w-3 h-3 text-on-surface-variant" />
              {job.location}
            </span>
          )}
          {job.salary_range && (
            <span className="bg-surface-container text-on-surface px-2.5 py-1 rounded-full text-[11px] font-medium flex items-center gap-1">
              <DollarSign className="w-3 h-3 text-on-surface-variant" />
              {job.salary_range}
            </span>
          )}
          <span className={`border text-[11px] font-semibold px-2.5 py-0.5 rounded-full ${sourceColorClass}`}>
            {job.source_board}
          </span>
          {job.posted_at && (
            <span className="flex items-center gap-1 text-[11px] text-on-surface-variant ml-auto font-medium">
              <Calendar className="w-3 h-3" />
              {formatDate(job.posted_at)}
            </span>
          )}
        </div>

        {/* Matched / Missing skills */}
        <div className="flex flex-wrap gap-1.5 mt-3.5">
          {job.matched_skills?.slice(0, 3).map(skill => (
            <span key={skill} className="skill-pill-green">{skill}</span>
          ))}
          {job.missing_skills?.slice(0, 1).map(skill => (
            <span key={skill} className="skill-pill-amber">{skill}</span>
          ))}
        </div>
      </div>

      {/* Card Action Buttons */}
      <div className="flex items-center gap-2.5 mt-5 pt-4 border-t border-surface-variant">
        {onEditResume && (
          <button
            id={`btn-card-edit-resume-${job.id}`}
            onClick={(e) => { e.stopPropagation(); onEditResume(job); }}
            className="flex-1 bg-primary text-on-primary hover:bg-primary-container rounded-full py-2.5 px-4 text-xs font-semibold flex items-center justify-center gap-1.5 shadow-elevation-1 hover:shadow-elevation-2 transition-all active:scale-95"
            title="Tailor your resume for this job"
          >
            <Sparkles className="w-3.5 h-3.5" /> Tailor
          </button>
        )}
        <a
          href={job.source_url}
          target="_blank"
          rel="noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="flex-1 bg-surface-container-lowest border border-outline text-on-surface hover:bg-surface-container rounded-full py-2.5 px-4 text-xs font-semibold flex items-center justify-center gap-1.5 transition-all active:scale-95"
        >
          Apply <ExternalLink className="w-3.5 h-3.5" />
        </a>
      </div>
    </div>
  );
};

export default JobCard;

