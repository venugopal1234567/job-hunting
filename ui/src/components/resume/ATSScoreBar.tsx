import React from 'react';
import { TrendingUp, TrendingDown, Minus, Cpu, Loader2, RefreshCw } from 'lucide-react';

interface ATSScoreBarProps {
  score: number | null;
  previousScore?: number | null;
  loading?: boolean;
  jobTitle?: string;
  pendingChanges?: number;
  onReanalyze?: () => void;
}

const ATSScoreBar: React.FC<ATSScoreBarProps> = ({
  score,
  previousScore,
  loading,
  jobTitle,
  pendingChanges = 0,
  onReanalyze,
}) => {
  const circumference = 2 * Math.PI * 28;
  const atsScore = score ?? 0;
  const hasScore = score !== null;
  const offset = hasScore ? circumference - (atsScore / 100) * circumference : circumference;

  const scoreColor = (s: number) => {
    if (s >= 80) return { stroke: '#34d399', text: 'text-emerald-400', bg: 'from-emerald-500/20 to-emerald-500/5', border: 'border-emerald-500/20' };
    if (s >= 60) return { stroke: '#fbbf24', text: 'text-amber-400', bg: 'from-amber-500/20 to-amber-500/5', border: 'border-amber-500/20' };
    return { stroke: '#f87171', text: 'text-red-400', bg: 'from-red-500/20 to-red-500/5', border: 'border-red-500/20' };
  };

  const colors = scoreColor(atsScore);

  const delta = previousScore !== null && previousScore !== undefined && score !== null
    ? score - previousScore
    : null;

  return (
    <div id="ats-score-bar" className={`flex items-center gap-4 px-4 py-2.5 rounded-xl border bg-gradient-to-r ${colors.bg} ${colors.border}`}>
      {/* Circular gauge */}
      <div className="relative flex-shrink-0">
        {loading ? (
          <div className="w-14 h-14 flex items-center justify-center">
            <Loader2 className="w-6 h-6 text-brand-400 animate-spin" />
          </div>
        ) : (
          <>
            <svg width="56" height="56" className="-rotate-90">
              <circle cx="28" cy="28" r="28" fill="none" stroke="rgba(255,255,255,0.06)" strokeWidth="5" />
              {hasScore && (
                <circle
                  cx="28" cy="28" r="28"
                  fill="none"
                  stroke={colors.stroke}
                  strokeWidth="5"
                  strokeDasharray={circumference}
                  strokeDashoffset={offset}
                  strokeLinecap="round"
                  className="transition-all duration-1000 gauge-ring"
                  style={{ '--target-offset': offset } as React.CSSProperties}
                />
              )}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              {hasScore ? (
                <>
                  <span className={`text-base font-bold leading-none ${colors.text}`}>{atsScore}</span>
                  <span className="text-[8px] text-gray-500 mt-0.5">ATS</span>
                </>
              ) : (
                <Cpu className="w-4 h-4 text-gray-600" />
              )}
            </div>
          </>
        )}
      </div>

      {/* Labels */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <p className="text-xs font-semibold text-white">
            {hasScore
              ? atsScore >= 80 ? 'Strong Match' : atsScore >= 60 ? 'Good Match' : 'Needs Work'
              : 'No ATS Score'
            }
          </p>
          {/* Delta */}
          {delta !== null && delta !== 0 && (
            <span className={`text-[10px] font-bold flex items-center gap-0.5 ${delta > 0 ? 'text-emerald-400' : 'text-red-400'}`}>
              {delta > 0 ? <TrendingUp className="w-3 h-3" /> : <TrendingDown className="w-3 h-3" />}
              {delta > 0 ? '+' : ''}{delta} pts
            </span>
          )}
          {delta === 0 && previousScore !== null && (
            <span className="text-[10px] text-gray-500 flex items-center gap-0.5">
              <Minus className="w-3 h-3" /> no change
            </span>
          )}
        </div>
        {jobTitle && (
          <p className="text-[10px] text-gray-500 truncate mt-0.5">For: {jobTitle}</p>
        )}
        {!hasScore && !loading && (
          <p className="text-[10px] text-gray-500 mt-0.5">Select a job to see ATS score</p>
        )}
        {pendingChanges > 0 && (
          <p className="text-[10px] text-amber-400 mt-0.5">
            ⚡ {pendingChanges} pending change{pendingChanges > 1 ? 's' : ''} — accept to improve score
          </p>
        )}
      </div>

      {/* Re-analyze button */}
      {onReanalyze && (
        <button
          id="btn-reanalyze-ats"
          onClick={onReanalyze}
          disabled={loading}
          title="Re-calculate ATS score"
          className="btn-ghost p-1.5 flex-shrink-0 text-gray-500 hover:text-brand-400"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
        </button>
      )}
    </div>
  );
};

export default ATSScoreBar;
