import React from 'react';
import { TrendingUp, TrendingDown, Minus, Cpu, Loader2, RefreshCw } from 'lucide-react';

interface ATSScoreBarProps {
  score: number | null;
  previousScore?: number | null;
  loading?: boolean;
  jobTitle?: string;
  onReanalyze?: () => void;
}

const ATSScoreBar: React.FC<ATSScoreBarProps> = ({
  score,
  previousScore,
  loading,
  jobTitle,
  onReanalyze,
}) => {
  const circumference = 2 * Math.PI * 28;
  const atsScore = score ?? 0;
  const hasScore = score !== null;
  const offset = hasScore ? circumference - (atsScore / 100) * circumference : circumference;

  const scoreColor = (s: number) => {
    if (s >= 80) return { stroke: '#4800b2', text: 'text-primary', bg: 'bg-primary-fixed/40', border: 'border-primary-fixed-dim/50' };
    if (s >= 60) return { stroke: '#9e4039', text: 'text-secondary', bg: 'bg-secondary-fixed/40', border: 'border-secondary-fixed-dim/50' };
    return { stroke: '#7a7488', text: 'text-on-surface-variant', bg: 'bg-surface-container', border: 'border-surface-variant' };
  };

  const colors = scoreColor(atsScore);

  const delta = previousScore !== null && previousScore !== undefined && score !== null
    ? score - previousScore
    : null;

  return (
    <div id="ats-score-bar" className={`flex items-center gap-3.5 px-4 py-2 rounded-2xl border ${colors.bg} ${colors.border}`}>
      {/* Circular gauge */}
      <div className="relative flex-shrink-0">
        {loading ? (
          <div className="w-12 h-12 flex items-center justify-center">
            <Loader2 className="w-5 h-5 text-primary animate-spin" />
          </div>
        ) : (
          <>
            <svg width="48" height="48" className="-rotate-90">
              <circle cx="24" cy="24" r="20" fill="none" stroke="#ededf2" strokeWidth="4" />
              {hasScore && (
                <circle
                  cx="24" cy="24" r="20"
                  fill="none"
                  stroke={colors.stroke}
                  strokeWidth="4"
                  strokeDasharray={2 * Math.PI * 20}
                  strokeDashoffset={2 * Math.PI * 20 - (atsScore / 100) * (2 * Math.PI * 20)}
                  strokeLinecap="round"
                  className="transition-all duration-1000"
                />
              )}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              {hasScore ? (
                <>
                  <span className={`text-sm font-bold leading-none font-headline ${colors.text}`}>{atsScore}</span>
                  <span className="text-[7px] text-on-surface-variant uppercase font-bold mt-0.5">ATS</span>
                </>
              ) : (
                <Cpu className="w-4 h-4 text-outline" />
              )}
            </div>
          </>
        )}
      </div>

      {/* Labels */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <p className="text-xs font-bold text-on-surface">
            {hasScore
              ? atsScore >= 80 ? 'Strong Match' : atsScore >= 60 ? 'Good Match' : 'Needs Work'
              : 'No ATS Score'
            }
          </p>
          {/* Delta */}
          {delta !== null && delta !== 0 && (
            <span className={`text-[10px] font-bold flex items-center gap-0.5 ${delta > 0 ? 'text-primary' : 'text-secondary'}`}>
              {delta > 0 ? <TrendingUp className="w-3 h-3" /> : <TrendingDown className="w-3 h-3" />}
              {delta > 0 ? '+' : ''}{delta} pts
            </span>
          )}
          {delta === 0 && previousScore !== null && (
            <span className="text-[10px] text-on-surface-variant flex items-center gap-0.5">
              <Minus className="w-3 h-3" /> no change
            </span>
          )}
        </div>
        {jobTitle && (
          <p className="text-[10px] text-on-surface-variant truncate mt-0.5">For: {jobTitle}</p>
        )}
      </div>

      {/* Reanalyze button */}
      {onReanalyze && (
        <button
          id="btn-reanalyze-ats"
          onClick={onReanalyze}
          disabled={loading}
          className="btn-ghost text-xs px-2.5 py-1 flex items-center gap-1 rounded-full text-primary hover:bg-primary-fixed/50"
          title="Recalculate ATS score against current resume"
        >
          <RefreshCw className={`w-3 h-3 ${loading ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline text-[11px] font-medium">Re-check</span>
        </button>
      )}
    </div>
  );
};

export default ATSScoreBar;
