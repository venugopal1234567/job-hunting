import React, { useEffect } from 'react';
import {
  X, ExternalLink, Cpu, Loader2, AlertCircle,
  CheckCircle2, AlertTriangle, MessageSquare, Lightbulb,
  MapPin, DollarSign, Calendar
} from 'lucide-react';
import { Job, ATSAnalysis } from '../../types';
import { useAtsAnalysis } from '../../hooks/useAtsAnalysis';

interface JobDetailModalProps {
  job: Job | null;
  onClose: () => void;
}

const JobDetailModal: React.FC<JobDetailModalProps> = ({ job, onClose }) => {
  const { analysis, loading, error, analyze, reset } = useAtsAnalysis();

  useEffect(() => {
    if (!job) {
      reset();
    }
  }, [job]);

  if (!job) return null;

  const circumference = 2 * Math.PI * 40;
  const atsScore = analysis?.ats_score ?? job.ats_score ?? 0;
  const hasAts = analysis !== null || job.ats_score !== null;
  const offset = hasAts ? circumference - (atsScore / 100) * circumference : circumference;

  const scoreColor = (s: number) => {
    if (s >= 80) return { stroke: '#34d399', text: 'text-emerald-400' };
    if (s >= 60) return { stroke: '#fbbf24', text: 'text-amber-400' };
    return { stroke: '#f87171', text: 'text-red-400' };
  };

  const colors = scoreColor(atsScore);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-end bg-black/60 backdrop-blur-sm animate-fade-in"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="w-full max-w-2xl h-full overflow-y-auto bg-surface-100 border-l border-white/5 animate-slide-up shadow-2xl">
        {/* Header */}
        <div className="sticky top-0 z-10 glass border-b border-white/5 px-6 py-4 flex items-start justify-between gap-4">
          <div className="flex-1">
            <h2 className="text-base font-bold text-white leading-snug">{job.title}</h2>
            <p className="text-sm text-gray-400 mt-0.5">{job.company}</p>
            <div className="flex flex-wrap items-center gap-3 mt-2">
              {job.location && (
                <span className="flex items-center gap-1 text-xs text-gray-500">
                  <MapPin className="w-3 h-3" /> {job.location}
                </span>
              )}
              {job.salary_range && (
                <span className="flex items-center gap-1 text-xs text-gray-500">
                  <DollarSign className="w-3 h-3" /> {job.salary_range}
                </span>
              )}
              {job.posted_at && (
                <span className="flex items-center gap-1 text-xs text-gray-500">
                  <Calendar className="w-3 h-3" /> {new Date(job.posted_at).toLocaleDateString()}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            <a
              href={job.source_url}
              target="_blank"
              rel="noreferrer"
              className="btn-primary text-xs font-semibold px-3 py-1.5 flex items-center gap-1.5 shadow-md shadow-brand-500/20 hover:scale-105 transition-all"
              id="btn-open-job-link"
            >
              Apply Now
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
            <button onClick={onClose} className="btn-ghost p-2" id="btn-close-job-modal">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* AI ATS Analysis Panel */}
          <div className="glass rounded-xl p-5">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Cpu className="w-4 h-4 text-brand-400" />
                <span className="text-sm font-semibold text-white">AI ATS Analysis</span>
                <span className="text-[10px] bg-brand-500/20 text-brand-300 border border-brand-500/30 px-1.5 py-0.5 rounded-full">
                  Gemma 4
                </span>
              </div>
              {!loading && (
                <button
                  id="btn-run-ats-analysis"
                  onClick={() => analyze(job.id)}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  <Cpu className="w-3 h-3" />
                  {analysis ? 'Re-analyze' : 'Analyze'}
                </button>
              )}
            </div>

            {loading && (
              <div className="flex flex-col items-center py-8 gap-3">
                <Loader2 className="w-8 h-8 text-brand-400 animate-spin" />
                <p className="text-sm text-gray-400">Gemma 4 is analyzing your resume match...</p>
                <p className="text-xs text-gray-600">This may take 30-60 seconds</p>
              </div>
            )}

            {error && !loading && (
              <div className="flex items-center gap-2 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
                <p className="text-xs text-red-300">{error}</p>
              </div>
            )}

            {!loading && !error && (analysis || job.ats_score !== null) && (
              <div className="space-y-5">
                {/* Score Gauge */}
                <div className="flex items-center gap-6">
                  <div className="relative flex-shrink-0">
                    <svg width="100" height="100" className="-rotate-90">
                      <circle cx="50" cy="50" r="40" fill="none" stroke="rgba(255,255,255,0.06)" strokeWidth="6" />
                      <circle
                        cx="50" cy="50" r="40"
                        fill="none"
                        stroke={colors.stroke}
                        strokeWidth="6"
                        strokeDasharray={circumference}
                        strokeDashoffset={offset}
                        strokeLinecap="round"
                        className="transition-all duration-1000 gauge-ring"
                        style={{ '--target-offset': offset } as React.CSSProperties}
                      />
                    </svg>
                    <div className="absolute inset-0 flex flex-col items-center justify-center rotate-0">
                      <span className={`text-2xl font-bold ${colors.text}`}>{atsScore}</span>
                      <span className="text-[9px] text-gray-500 -mt-1">ATS Score</span>
                    </div>
                  </div>
                  <div className="flex-1 space-y-2">
                    <div>
                      <p className="section-header">Matched Skills</p>
                      <div className="flex flex-wrap gap-1.5">
                        {(analysis?.match_breakdown?.matched_skills ?? job.matched_skills ?? []).map(s => (
                          <span key={s} className="skill-pill-green flex items-center gap-1">
                            <CheckCircle2 className="w-2.5 h-2.5" /> {s}
                          </span>
                        ))}
                        {(analysis?.match_breakdown?.matched_skills ?? job.matched_skills ?? []).length === 0 && (
                          <span className="text-xs text-gray-600">None detected</span>
                        )}
                      </div>
                    </div>
                    <div>
                      <p className="section-header">Missing Skills</p>
                      <div className="flex flex-wrap gap-1.5">
                        {(analysis?.match_breakdown?.missing_skills ?? job.missing_skills ?? []).map(s => (
                          <span key={s} className="skill-pill-amber flex items-center gap-1">
                            <AlertTriangle className="w-2.5 h-2.5" /> {s}
                          </span>
                        ))}
                        {(analysis?.match_breakdown?.missing_skills ?? job.missing_skills ?? []).length === 0 && (
                          <span className="text-xs text-gray-600">Great match!</span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>

                {/* Actionable Suggestions */}
                {analysis?.actionable_suggestions && analysis.actionable_suggestions.length > 0 && (
                  <div>
                    <p className="section-header flex items-center gap-1.5">
                      <Lightbulb className="w-3.5 h-3.5 text-amber-400" />
                      Resume Improvement Suggestions
                    </p>
                    <ul className="space-y-2">
                      {analysis.actionable_suggestions.map((s, i) => (
                        <li key={i} className="flex gap-2 text-xs text-gray-300 bg-surface-200 rounded-lg p-3">
                          <span className="text-amber-400 font-bold flex-shrink-0">{i + 1}.</span>
                          {s}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Gap Questions */}
                {analysis?.gap_questions && analysis.gap_questions.length > 0 && (
                  <div>
                    <p className="section-header flex items-center gap-1.5">
                      <MessageSquare className="w-3.5 h-3.5 text-brand-400" />
                      Interview Prep Questions
                    </p>
                    <div className="space-y-2.5">
                      {analysis.gap_questions.map((q, i) => (
                        <div key={i} className="bg-surface-200 rounded-lg p-3 border border-white/5">
                          <span className="skill-pill-amber mb-2 inline-flex">{q.skill}</span>
                          <p className="text-xs text-gray-300 mt-1.5">{q.question}</p>
                          <textarea
                            placeholder="Your notes..."
                            className="input-field w-full mt-2 text-xs resize-none"
                            rows={2}
                            id={`gap-question-notes-${i}`}
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {!loading && !error && !analysis && job.ats_score === null && (
              <div className="text-center py-8">
                <Cpu className="w-8 h-8 text-gray-600 mx-auto mb-2" />
                <p className="text-sm text-gray-500">Click "Analyze" to run AI ATS analysis</p>
                <p className="text-xs text-gray-600 mt-1">Requires an active resume to be uploaded</p>
              </div>
            )}
          </div>

          {/* Job Description */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <p className="section-header mb-0">Job Description</p>
              <a
                href={job.source_url}
                target="_blank"
                rel="noreferrer"
                className="text-xs text-brand-400 hover:text-brand-300 font-medium flex items-center gap-1"
              >
                View original post on {job.source_board} <ExternalLink className="w-3 h-3" />
              </a>
            </div>
            <div className="glass rounded-xl p-5 border border-white/5">
              {formatNeatDescription(job.description)}
            </div>
          </div>

          {/* Bottom Apply Banner */}
          <div className="bg-gradient-to-r from-brand-600/30 to-purple-600/30 border border-brand-500/30 rounded-xl p-4 flex flex-col sm:flex-row items-center justify-between gap-4 mt-6">
            <div>
              <p className="text-xs text-brand-300 font-semibold uppercase tracking-wider">Ready to apply?</p>
              <p className="text-sm font-bold text-white mt-0.5">{job.title} at {job.company}</p>
            </div>
            <a
              href={job.source_url}
              target="_blank"
              rel="noreferrer"
              className="btn-primary text-sm font-semibold px-5 py-2.5 flex items-center gap-2 shadow-lg shadow-brand-500/30 hover:scale-105 transition-all w-full sm:w-auto justify-center"
            >
              Apply on {job.source_board}
              <ExternalLink className="w-4 h-4" />
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

const formatNeatDescription = (desc: string) => {
  if (!desc) return null;

  // Clean anti-bot spam text & HTML entities
  let cleaned = desc
    .replace(/Please mention the word \*\*[^*]+\*\*[\s\S]*?they're human\./gi, '')
    .replace(/Please mention the word [A-Z0-9_*-]+[\s\S]*?spam applicants\./gi, '')
    .replace(/#RMTAz[A-Za-z0-9]+/g, '')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#039;/g, "'");

  const lines = cleaned.split(/\n+/).map(l => l.trim()).filter(Boolean);

  return (
    <div className="space-y-3 text-sm text-gray-300 leading-relaxed">
      {lines.map((line, idx) => {
        const isHeader = /^(Requirements|Responsibilities|About Us|About the Role|What You'll Do|Who You Are|Qualifications|Benefits|Tech Stack|Our Stack):?$/i.test(line) ||
          (line.length < 50 && line.endsWith(':'));

        const isBullet = line.startsWith('•') || line.startsWith('-') || line.startsWith('*');

        if (isHeader) {
          return (
            <h4 key={idx} className="text-sm font-bold text-white mt-4 border-b border-white/10 pb-1 flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-brand-400"></span>
              {line.replace(/:$/, '')}
            </h4>
          );
        }

        if (isBullet) {
          return (
            <div key={idx} className="flex items-start gap-2.5 pl-2 text-xs sm:text-sm text-gray-300">
              <span className="text-brand-400 mt-1 font-bold flex-shrink-0">•</span>
              <span>{line.replace(/^[-•*]\s*/, '')}</span>
            </div>
          );
        }

        return (
          <p key={idx} className="text-xs sm:text-sm text-gray-300 leading-relaxed">
            {line}
          </p>
        );
      })}
    </div>
  );
};

export default JobDetailModal;
