import React from 'react';
import { Sparkles, RotateCcw, Download, PanelRightClose, PanelRight, CheckCircle2 } from 'lucide-react';
import { Job } from '../../../types';
import ATSScoreBar from '../ATSScoreBar';

interface ResumeEditorHeaderProps {
  selectedJob?: Job | null;
  atsScore: number | null;
  prevAtsScore: number | null;
  atsLoading: boolean;
  onReanalyzeATS?: () => void;
  saveMessage: string;
  activeSubTab: 'editor' | 'pdf';
  onSubTabChange: (tab: 'editor' | 'pdf') => void;
  fitToSinglePage: boolean;
  onToggleFitToSinglePage: () => void;
  isSaving: boolean;
  onRevert: () => void;
  exportingPDF: boolean;
  onExportPDF: () => void;
  chatVisible: boolean;
  onToggleChat: () => void;
  onUploadClick: () => void;
}

export const ResumeEditorHeader: React.FC<ResumeEditorHeaderProps> = ({
  selectedJob,
  atsScore,
  prevAtsScore,
  atsLoading,
  onReanalyzeATS,
  saveMessage,
  activeSubTab,
  onSubTabChange,
  fitToSinglePage,
  onToggleFitToSinglePage,
  isSaving,
  onRevert,
  exportingPDF,
  onExportPDF,
  chatVisible,
  onToggleChat,
}) => {
  return (
    <div className="glass rounded-xl border border-white/10 p-3 mb-4 no-print shadow-xl">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {/* ATS Score Bar */}
        <div className="flex-1 min-w-[260px]">
          <ATSScoreBar
            score={atsScore}
            previousScore={prevAtsScore}
            loading={atsLoading}
            jobTitle={selectedJob?.title}
            onReanalyze={selectedJob ? onReanalyzeATS : undefined}
          />
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {/* Status indicator */}
          {saveMessage && (
            <span className="flex items-center gap-1 text-xs text-emerald-400 font-medium animate-fade-in mr-1">
              <CheckCircle2 className="w-3.5 h-3.5" /> {saveMessage}
            </span>
          )}

          {/* Sub-tabs */}
          <div className="flex items-center bg-surface-300 rounded-lg p-1 border border-white/5">
            <button
              onClick={() => onSubTabChange('editor')}
              className={`px-3 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-all ${
                activeSubTab === 'editor'
                  ? 'bg-indigo-600 text-white shadow-md'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              ✏️ Visual Canvas
            </button>
            <button
              onClick={() => onSubTabChange('pdf')}
              className={`px-3 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-all ${
                activeSubTab === 'pdf'
                  ? 'bg-indigo-600 text-white shadow-md'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              📄 Original PDF
            </button>
          </div>

          {/* Fit to page */}
          <button
            id="btn-fit-single-page"
            onClick={onToggleFitToSinglePage}
            className={`text-xs px-3 py-1.5 rounded-lg flex items-center gap-1.5 font-medium transition-all border ${
              fitToSinglePage
                ? 'bg-indigo-600 text-white border-indigo-400 shadow-md shadow-indigo-600/30'
                : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
            }`}
            title="Fit resume layout onto 1 single page"
          >
            <Sparkles className="w-3.5 h-3.5 text-yellow-300" />
            Fit to page {fitToSinglePage ? '✓' : ''}
          </button>

          {/* Revert */}
          <button
            id="btn-revert-resume"
            onClick={onRevert}
            disabled={isSaving}
            className="px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/20 text-xs font-medium flex items-center gap-1.5 disabled:opacity-40 transition-all"
            title="Revert all edits back to original resume text"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            Revert
          </button>

          {/* Export to PDF */}
          <button
            id="btn-export-pdf"
            onClick={onExportPDF}
            disabled={exportingPDF}
            className="px-3.5 py-1.5 rounded-lg bg-gradient-to-r from-indigo-600 to-blue-600 hover:from-indigo-500 hover:to-blue-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-600/30 disabled:opacity-50 transition-all"
            title="Export clean PDF file directly via backend API"
          >
            <Download className="w-3.5 h-3.5 text-white" />
            Export to PDF
          </button>

          {/* AI Resume Coach Toggle */}
          <button
            id="btn-toggle-chat"
            onClick={onToggleChat}
            className={`p-2 px-3 rounded-lg text-xs font-medium border flex items-center gap-1.5 transition-all ${
              chatVisible
                ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40'
                : 'bg-surface-200 text-gray-300 border-white/5 hover:bg-surface-300'
            }`}
            title={chatVisible ? 'Hide AI Resume Coach' : 'Show AI Resume Coach'}
          >
            {chatVisible ? <PanelRightClose className="w-4 h-4" /> : <PanelRight className="w-4 h-4" />}
            <span>AI Resume Coach</span>
          </button>
        </div>
      </div>
    </div>
  );
};
