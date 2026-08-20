import React, { useEffect } from 'react';
import {
  X, ExternalLink, Cpu, Loader2, AlertCircle,
  CheckCircle2, AlertTriangle, MessageSquare, Lightbulb,
  MapPin, DollarSign, Calendar, Sparkles
} from 'lucide-react';
import { Job } from '../../types';
import { useAtsAnalysis } from '../../hooks/useAtsAnalysis';

interface JobDetailModalProps {
  job: Job | null;
  onClose: () => void;
  onEditResume?: (job: Job) => void;
}

const JobDetailModal: React.FC<JobDetailModalProps> = ({ job, onClose, onEditResume }) => {
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
    if (s >= 80) return { stroke: '#4800b2', text: 'text-primary' };
    if (s >= 60) return { stroke: '#9e4039', text: 'text-secondary' };
    return { stroke: '#7a7488', text: 'text-on-surface-variant' };
  };

  const colors = scoreColor(atsScore);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-end bg-inverse-surface/30 backdrop-blur-sm animate-fade-in"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="w-full max-w-2xl h-full overflow-y-auto bg-surface-container-lowest border-l border-surface-variant animate-slide-up shadow-elevation-3 flex flex-col justify-between">
        <div>
          {/* Header */}
          <div className="sticky top-0 z-10 bg-surface-container-lowest/90 backdrop-blur-xl border-b border-surface-variant px-6 py-5 flex items-start justify-between gap-4">
            <div className="flex-1">
              <h2 className="text-lg font-bold font-headline text-on-surface leading-snug">{job.title}</h2>
              <p className="text-sm font-medium text-on-surface-variant mt-0.5">{job.company}</p>
              <div className="flex flex-wrap items-center gap-2.5 mt-3">
                {job.location && (
                  <span className="bg-surface-container text-on-surface px-2.5 py-1 rounded-full text-xs font-medium flex items-center gap-1">
                    <MapPin className="w-3.5 h-3.5 text-on-surface-variant" /> {job.location}
                  </span>
                )}
                {job.salary_range && (
                  <span className="bg-surface-container text-on-surface px-2.5 py-1 rounded-full text-xs font-medium flex items-center gap-1">
                    <DollarSign className="w-3.5 h-3.5 text-on-surface-variant" /> {job.salary_range}
                  </span>
                )}
                {job.posted_at && (
                  <span className="bg-surface-container text-on-surface px-2.5 py-1 rounded-full text-xs font-medium flex items-center gap-1">
                    <Calendar className="w-3.5 h-3.5 text-on-surface-variant" /> {new Date(job.posted_at).toLocaleDateString()}
                  </span>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2 flex-shrink-0">
              <button onClick={onClose} className="p-2 text-on-surface-variant hover:text-primary rounded-full hover:bg-surface-container transition-colors" id="btn-close-job-modal">
                <X className="w-5 h-5" />
              </button>
            </div>
          </div>

          <div className="p-6 space-y-6">
            {/* AI ATS Analysis Panel */}
            <div className="bg-surface-container-low border border-surface-variant rounded-2xl p-5 shadow-elevation-1">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 rounded-full bg-primary-fixed text-primary">
                    <Sparkles className="w-4 h-4" />
                  </div>
                  <span className="text-sm font-bold font-headline text-on-surface">AI ATS Analysis</span>
                  <span className="text-[11px] bg-primary-fixed text-on-primary-fixed border border-primary-fixed-dim/50 px-2 py-0.5 rounded-full font-medium">
                    Gemma 4
                  </span>
                </div>
                {!loading && (
                  <button
                    id="btn-run-ats-analysis"
                    onClick={() => analyze(job.id, undefined, true)}
                    className="btn-primary text-xs py-1.5 px-4 flex items-center gap-1.5"
                  >
                    <Sparkles className="w-3.5 h-3.5" />
                    {analysis ? 'Re-analyze' : 'Analyze'}
                  </button>
                )}
              </div>

              {loading && (
                <div className="flex flex-col items-center py-8 gap-3">
                  <Loader2 className="w-8 h-8 text-primary animate-spin" />
                  <p className="text-sm font-medium text-on-surface">Gemma 4 is analyzing your resume match...</p>
                  <p className="text-xs text-on-surface-variant">This may take 30-60 seconds</p>
                </div>
              )}

              {error && !loading && (
                <div className="flex items-center gap-2 p-3 bg-error-container text-on-error-container border border-error/20 rounded-xl">
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  <p className="text-xs">{error}</p>
                </div>
              )}

              {!loading && !error && (analysis || job.ats_score !== null) && (
                <div className="space-y-5">
                  {/* Score Gauge */}
                  <div className="flex items-center gap-6 bg-surface-container-lowest p-4 rounded-xl border border-surface-variant">
                    <div className="relative flex-shrink-0">
                      <svg width="90" height="90" className="-rotate-90">
                        <circle cx="45" cy="45" r="36" fill="none" stroke="#ededf2" strokeWidth="6" />
                        <circle
                          cx="45" cy="45" r="36"
                          fill="none"
                          stroke={colors.stroke}
                          strokeWidth="6"
                          strokeDasharray={circumference}
                          strokeDashoffset={offset}
                          strokeLinecap="round"
                          className="transition-all duration-1000"
                        />
                      </svg>
                      <div className="absolute inset-0 flex flex-col items-center justify-center rotate-0">
                        <span className={`text-2xl font-bold font-headline ${colors.text}`}>{atsScore}</span>
                        <span className="text-[9px] text-on-surface-variant -mt-1 font-semibold">ATS Match</span>
                      </div>
                    </div>
                    <div className="flex-1 space-y-3">
                      <div>
                        <p className="text-[11px] font-semibold uppercase tracking-wider text-on-surface-variant mb-1.5">
                          Matched Skills
                        </p>
                        <div className="flex flex-wrap gap-1.5">
                          {(analysis?.match_breakdown?.matched_skills ?? job.matched_skills ?? []).map(s => (
                            <span key={s} className="skill-pill-green flex items-center gap-1">
                              <CheckCircle2 className="w-3 h-3" /> {s}
                            </span>
                          ))}
                          {(analysis?.match_breakdown?.matched_skills ?? job.matched_skills ?? []).length === 0 && (
                            <span className="text-xs text-on-surface-variant">None detected</span>
                          )}
                        </div>
                      </div>
                      <div>
                        <p className="text-[11px] font-semibold uppercase tracking-wider text-on-surface-variant mb-1.5">
                          Missing Skills
                        </p>
                        <div className="flex flex-wrap gap-1.5">
                          {(analysis?.match_breakdown?.missing_skills ?? job.missing_skills ?? []).map(s => (
                            <span key={s} className="skill-pill-amber flex items-center gap-1">
                              <AlertTriangle className="w-3 h-3" /> {s}
                            </span>
                          ))}
                          {(analysis?.match_breakdown?.missing_skills ?? job.missing_skills ?? []).length === 0 && (
                            <span className="text-xs text-on-surface-variant">Great match!</span>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Actionable Suggestions */}
                  {analysis?.actionable_suggestions && analysis.actionable_suggestions.length > 0 && (
                    <div>
                      <p className="text-xs uppercase font-bold tracking-wider text-on-surface-variant mb-2.5 flex items-center gap-1.5">
                        <Lightbulb className="w-4 h-4 text-secondary" />
                        Resume Improvement Suggestions
                      </p>
                      <ul className="space-y-2">
                        {analysis.actionable_suggestions.map((s, i) => (
                          <li key={i} className="flex gap-2 text-xs text-on-surface bg-surface-container-lowest border border-surface-variant rounded-xl p-3.5">
                            <span className="text-secondary font-bold flex-shrink-0">{i + 1}.</span>
                            <span>{s}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {/* Gap Questions */}
                  {analysis?.gap_questions && analysis.gap_questions.length > 0 && (
                    <div>
                      <p className="text-xs uppercase font-bold tracking-wider text-on-surface-variant mb-2.5 flex items-center gap-1.5">
                        <MessageSquare className="w-4 h-4 text-primary" />
                        Interview Prep Questions
                      </p>
                      <div className="space-y-2.5">
                        {analysis.gap_questions.map((q, i) => (
                          <div key={i} className="bg-surface-container-lowest rounded-xl p-3.5 border border-surface-variant">
                            <span className="skill-pill-amber mb-2 inline-flex">{q.skill}</span>
                            <p className="text-xs text-on-surface mt-1.5">{q.question}</p>
                            <textarea
                              placeholder="Your notes..."
                              className="input-field w-full mt-2 text-xs resize-none rounded-xl"
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
                  <div className="p-3 rounded-full bg-surface-container inline-flex text-on-surface-variant mb-2">
                    <Sparkles className="w-6 h-6" />
                  </div>
                  <p className="text-sm font-semibold text-on-surface">Click "Analyze" to run AI ATS analysis</p>
                  <p className="text-xs text-on-surface-variant mt-1">Requires an active resume to be uploaded</p>
                </div>
              )}
            </div>

            {/* Job Description */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <p className="text-xs font-bold uppercase tracking-wider text-on-surface-variant">Job Description</p>
                <a
                  href={job.source_url}
                  target="_blank"
                  rel="noreferrer"
                  className="text-xs text-primary hover:underline font-semibold flex items-center gap-1"
                >
                  View original on {job.source_board} <ExternalLink className="w-3 h-3" />
                </a>
              </div>
              <div className="bg-surface-container-lowest rounded-2xl p-6 border border-surface-variant shadow-elevation-1">
                {formatNeatDescription(job.description)}
              </div>
            </div>
          </div>
        </div>

        {/* Bottom Sticky Action Banner */}
        <div className="p-6 bg-surface-container-low border-t border-surface-variant flex flex-col sm:flex-row items-center justify-between gap-4">
          <div>
            <p className="text-xs text-secondary font-bold uppercase tracking-wider">Ready to apply?</p>
            <p className="text-sm font-bold text-on-surface mt-0.5">{job.title} at {job.company}</p>
          </div>
          <div className="flex gap-2.5 w-full sm:w-auto">
            {onEditResume && (
              <button
                id="btn-edit-resume-banner"
                onClick={() => { onEditResume(job); onClose(); }}
                className="btn-primary text-xs flex items-center gap-2 w-full sm:w-auto justify-center"
              >
                <Sparkles className="w-4 h-4" />
                Tailor Resume
              </button>
            )}
            <a
              href={job.source_url}
              target="_blank"
              rel="noreferrer"
              className="btn-outline py-2.5 px-5 flex items-center gap-2 text-xs justify-center w-full sm:w-auto"
            >
              Apply on {job.source_board}
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

const formatNeatDescription = (desc: string) => {
  if (!desc) return null;

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
    <div className="space-y-3 text-sm text-on-surface leading-relaxed">
      {lines.map((line, idx) => {
        const isHeader = /^(Requirements|Responsibilities|About Us|About the Role|What You'll Do|Who You Are|Qualifications|Benefits|Tech Stack|Our Stack):?$/i.test(line) ||
          (line.length < 50 && line.endsWith(':'));

        const isBullet = line.startsWith('•') || line.startsWith('-') || line.startsWith('*');

        if (isHeader) {
          return (
            <h4 key={idx} className="text-sm font-bold font-headline text-on-surface mt-4 border-b border-surface-variant pb-1 flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-primary"></span>
              {line.replace(/:$/, '')}
            </h4>
          );
        }

        if (isBullet) {
          return (
            <div key={idx} className="flex items-start gap-2.5 pl-2 text-xs sm:text-sm text-on-surface">
              <span className="text-primary mt-1 font-bold flex-shrink-0">•</span>
              <span>{line.replace(/^[-•*]\s*/, '')}</span>
            </div>
          );
        }

        return (
          <p key={idx} className="text-xs sm:text-sm text-on-surface-variant leading-relaxed">
            {line}
          </p>
        );
      })}
    </div>
  );
};

export default JobDetailModal;

